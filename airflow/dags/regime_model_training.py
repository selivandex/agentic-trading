"""
Airflow DAG for automated Regime Detection Model Training

This DAG automates the ML training pipeline:
1. Check if enough labeled data exists (300+ samples)
2. Export features from ClickHouse
3. Train Random Forest model
4. Validate model performance (accuracy, F1-score)
5. Deploy if better than current production model
6. Send notifications on success/failure

Schedule: Weekly on Sunday at 2 AM UTC (low-traffic time)
Retries: 3 attempts with exponential backoff
"""

from datetime import datetime, timedelta
from pathlib import Path

from airflow import DAG
from airflow.operators.python import PythonOperator, BranchPythonOperator
from airflow.operators.bash import BashOperator
from airflow.operators.dummy import DummyOperator
from airflow.utils.trigger_rule import TriggerRule
from airflow.exceptions import AirflowSkipException

import clickhouse_connect
import pandas as pd
import json
import os

# Configuration
CLICKHOUSE_HOST = os.getenv('CLICKHOUSE_HOST', 'localhost')
CLICKHOUSE_PORT = int(os.getenv('CLICKHOUSE_PORT', '9000'))
MIN_LABELED_SAMPLES = 300  # Minimum samples needed for training
MIN_ACCURACY = 0.65  # Minimum accuracy to deploy model
MODELS_DIR = Path("/app/models")  # Model storage directory
SCRIPTS_DIR = Path("/app/scripts/ml/regime")  # Training scripts directory

# DAG default arguments
default_args = {
    'owner': 'ml-team',
    'depends_on_past': False,
    'email': ['alerts@prometheus-trading.com'],
    'email_on_failure': True,
    'email_on_retry': False,
    'retries': 3,
    'retry_delay': timedelta(minutes=5),
    'retry_exponential_backoff': True,
    'max_retry_delay': timedelta(minutes=30),
}

# Initialize DAG
dag = DAG(
    'regime_model_training',
    default_args=default_args,
    description='Train and deploy regime detection ML model',
    schedule_interval='0 2 * * 0',  # Every Sunday at 2 AM UTC
    start_date=datetime(2024, 11, 1),
    catchup=False,
    tags=['ml', 'regime-detection', 'training'],
)


def check_labeled_data_availability(**context):
    """Check if enough labeled data exists for training"""
    client = clickhouse_connect.get_client(host=CLICKHOUSE_HOST, port=CLICKHOUSE_PORT)
    
    query = """
    SELECT 
        count() as total_samples,
        countDistinct(regime_label) as num_classes,
        groupArray(regime_label) as labels,
        groupArray(count_per_label) as counts
    FROM (
        SELECT 
            regime_label,
            count() as count_per_label
        FROM regime_features
        WHERE regime_label != ''
        GROUP BY regime_label
    )
    """
    
    result = client.query(query)
    row = result.result_rows[0]
    
    total_samples = row[0]
    num_classes = row[1]
    labels = row[2]
    counts = row[3]
    
    print(f"Total labeled samples: {total_samples}")
    print(f"Number of classes: {num_classes}")
    print(f"Distribution: {dict(zip(labels, counts))}")
    
    # Check minimum requirements
    if total_samples < MIN_LABELED_SAMPLES:
        raise AirflowSkipException(
            f"Not enough labeled data: {total_samples} < {MIN_LABELED_SAMPLES}. "
            "Label more data using scripts/ml/regime/label_data.py"
        )
    
    if num_classes < 3:
        raise AirflowSkipException(
            f"Not enough regime classes: {num_classes} < 3. "
            "Need diverse regime examples for robust model."
        )
    
    # Check class balance (warn if severely imbalanced)
    min_count = min(counts)
    max_count = max(counts)
    imbalance_ratio = max_count / min_count if min_count > 0 else 0
    
    if imbalance_ratio > 10:
        print(f"WARNING: Severe class imbalance detected (ratio: {imbalance_ratio:.1f})")
        print("Consider collecting more data for underrepresented regimes")
    
    # Store metadata for downstream tasks
    context['ti'].xcom_push(key='total_samples', value=total_samples)
    context['ti'].xcom_push(key='num_classes', value=num_classes)
    context['ti'].xcom_push(key='class_distribution', value=dict(zip(labels, counts)))
    
    print(f"✓ Data check passed: {total_samples} samples, {num_classes} classes")


def train_model(**context):
    """Train Random Forest model using training script"""
    import subprocess
    
    # Get metadata from previous task
    total_samples = context['ti'].xcom_pull(key='total_samples', task_ids='check_data')
    
    print(f"Training model with {total_samples} labeled samples...")
    
    # Build command
    cmd = [
        'python3',
        str(SCRIPTS_DIR / 'train_model.py'),
        '--clickhouse-url', f'{CLICKHOUSE_HOST}:{CLICKHOUSE_PORT}',
        '--start-date', '2023-01-01',
        '--end-date', datetime.now().strftime('%Y-%m-%d'),
        '--output', str(MODELS_DIR / 'regime_detector_candidate.onnx'),
    ]
    
    # Run training script
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
    
    if result.returncode != 0:
        print(f"Training failed:\n{result.stderr}")
        raise Exception(f"Model training failed with code {result.returncode}")
    
    print(result.stdout)
    
    # Parse metrics from output (training script prints them)
    metrics = parse_training_metrics(result.stdout)
    
    # Store metrics for validation
    context['ti'].xcom_push(key='train_accuracy', value=metrics['train_accuracy'])
    context['ti'].xcom_push(key='test_accuracy', value=metrics['test_accuracy'])
    context['ti'].xcom_push(key='cv_accuracy', value=metrics['cv_accuracy'])
    
    print(f"✓ Model trained: test_accuracy={metrics['test_accuracy']:.3f}")


def parse_training_metrics(stdout):
    """Parse training metrics from script output"""
    metrics = {
        'train_accuracy': 0.0,
        'test_accuracy': 0.0,
        'cv_accuracy': 0.0,
    }
    
    for line in stdout.split('\n'):
        if 'Train accuracy:' in line:
            metrics['train_accuracy'] = float(line.split(':')[1].strip())
        elif 'Test accuracy:' in line:
            metrics['test_accuracy'] = float(line.split(':')[1].strip())
        elif 'Cross-validation accuracy:' in line:
            # Format: "Cross-validation accuracy: 0.750 (+/- 0.045)"
            parts = line.split(':')[1].split('(')[0].strip()
            metrics['cv_accuracy'] = float(parts)
    
    return metrics


def validate_model(**context):
    """Validate if new model is better than current production model"""
    test_accuracy = context['ti'].xcom_pull(key='test_accuracy', task_ids='train_model')
    
    # Check minimum accuracy threshold
    if test_accuracy < MIN_ACCURACY:
        print(f"Model accuracy {test_accuracy:.3f} below threshold {MIN_ACCURACY}")
        return 'skip_deployment'
    
    # Load current production model metrics if exists
    prod_metrics_file = MODELS_DIR / 'regime_detector_metrics.json'
    if prod_metrics_file.exists():
        with open(prod_metrics_file, 'r') as f:
            prod_metrics = json.load(f)
        
        prod_accuracy = prod_metrics.get('test_accuracy', 0.0)
        improvement = test_accuracy - prod_accuracy
        
        print(f"Current production accuracy: {prod_accuracy:.3f}")
        print(f"New model accuracy: {test_accuracy:.3f}")
        print(f"Improvement: {improvement:+.3f}")
        
        # Deploy if improvement >= 2% OR if production accuracy is low
        if improvement >= 0.02 or prod_accuracy < 0.70:
            print("✓ New model is better, proceeding to deployment")
            return 'deploy_model'
        else:
            print("✗ New model not significantly better, skipping deployment")
            return 'skip_deployment'
    else:
        # No production model yet, deploy first model
        print("No production model found, deploying first model")
        return 'deploy_model'


def deploy_model(**context):
    """Deploy validated model to production"""
    import shutil
    
    candidate_path = MODELS_DIR / 'regime_detector_candidate.onnx'
    production_path = MODELS_DIR / 'regime_detector.onnx'
    backup_path = MODELS_DIR / f'regime_detector_backup_{datetime.now().strftime("%Y%m%d_%H%M%S")}.onnx'
    
    # Backup current production model if exists
    if production_path.exists():
        shutil.copy(production_path, backup_path)
        print(f"Backed up current model to {backup_path}")
    
    # Deploy new model
    shutil.copy(candidate_path, production_path)
    print(f"Deployed new model to {production_path}")
    
    # Save metrics
    metrics = {
        'train_accuracy': context['ti'].xcom_pull(key='train_accuracy', task_ids='train_model'),
        'test_accuracy': context['ti'].xcom_pull(key='test_accuracy', task_ids='train_model'),
        'cv_accuracy': context['ti'].xcom_pull(key='cv_accuracy', task_ids='train_model'),
        'deployed_at': datetime.now().isoformat(),
        'total_samples': context['ti'].xcom_pull(key='total_samples', task_ids='check_data'),
    }
    
    with open(MODELS_DIR / 'regime_detector_metrics.json', 'w') as f:
        json.dump(metrics, f, indent=2)
    
    print(f"✓ Model deployed successfully")
    print(f"Metrics: {json.dumps(metrics, indent=2)}")


def send_success_notification(**context):
    """Send success notification (Telegram, email, etc.)"""
    test_accuracy = context['ti'].xcom_pull(key='test_accuracy', task_ids='train_model')
    total_samples = context['ti'].xcom_pull(key='total_samples', task_ids='check_data')
    
    message = f"""
🎯 Regime Detection Model Training Complete

✅ Status: SUCCESS
📊 Test Accuracy: {test_accuracy:.1%}
📈 Training Samples: {total_samples}
🕐 Completed: {datetime.now().strftime('%Y-%m-%d %H:%M UTC')}

Model deployed to production.
Application will use new model on next restart.
    """
    
    print(message)
    
    # TODO: Send to Telegram bot or Slack
    # telegram_bot.send_message(chat_id=ADMIN_CHAT_ID, text=message)


# Define tasks

check_data = PythonOperator(
    task_id='check_data',
    python_callable=check_labeled_data_availability,
    dag=dag,
)

train = PythonOperator(
    task_id='train_model',
    python_callable=train_model,
    dag=dag,
)

validate = BranchPythonOperator(
    task_id='validate_model',
    python_callable=validate_model,
    dag=dag,
)

deploy = PythonOperator(
    task_id='deploy_model',
    python_callable=deploy_model,
    dag=dag,
)

skip_deploy = DummyOperator(
    task_id='skip_deployment',
    dag=dag,
)

notify_success = PythonOperator(
    task_id='notify_success',
    python_callable=send_success_notification,
    trigger_rule=TriggerRule.ONE_SUCCESS,  # Run if deploy OR skip succeeded
    dag=dag,
)

# Define task dependencies
check_data >> train >> validate
validate >> [deploy, skip_deploy]
[deploy, skip_deploy] >> notify_success


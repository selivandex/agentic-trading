#!/usr/bin/env python3
"""
Train regime classification model using historical features from ClickHouse.

Usage:
    python train_model.py --clickhouse-url localhost:9000 --output models/regime_detector.onnx
"""

import argparse
import pandas as pd
import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import train_test_split, cross_val_score
from sklearn.metrics import classification_report, confusion_matrix
import skl2onnx
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import FloatTensorType
import clickhouse_connect

# Feature names (must match Features struct order in Go code)
FEATURE_NAMES = [
    'atr_14', 'atr_pct', 'bb_width', 'historical_vol',
    'adx', 'ema_9', 'ema_21', 'ema_55', 'ema_200',
    'higher_highs_count', 'lower_lows_count',
    'volume_24h', 'volume_change_pct', 'volume_price_divergence',
    'support_breaks', 'resistance_breaks', 'consolidation_periods',
    'btc_dominance', 'correlation_tightness',
    'funding_rate', 'funding_rate_ma', 'liquidations_24h'
]

def load_training_data(clickhouse_url, start_date, end_date):
    """Load labeled features from ClickHouse."""
    host, port = clickhouse_url.split(':')
    client = clickhouse_connect.get_client(host=host, port=int(port))
    
    query = f"""
    SELECT {', '.join(FEATURE_NAMES)}, regime_label
    FROM regime_features
    WHERE regime_label != ''
      AND timestamp >= '{start_date}'
      AND timestamp <= '{end_date}'
    ORDER BY timestamp
    """
    
    result = client.query(query)
    df = pd.DataFrame(result.result_rows, columns=FEATURE_NAMES + ['regime_label'])
    
    print(f"Loaded {len(df)} labeled samples")
    print(f"Class distribution:\n{df['regime_label'].value_counts()}")
    
    return df

def train_model(df, output_path):
    """Train Random Forest classifier and export to ONNX."""
    X = df[FEATURE_NAMES].values
    y = df['regime_label'].values
    
    # Split data
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )
    
    print(f"Training set: {len(X_train)}, Test set: {len(X_test)}")
    
    # Train model
    print("Training Random Forest classifier...")
    model = RandomForestClassifier(
        n_estimators=200,
        max_depth=10,
        min_samples_split=10,
        min_samples_leaf=5,
        random_state=42,
        n_jobs=-1
    )
    
    model.fit(X_train, y_train)
    
    # Evaluate
    train_score = model.score(X_train, y_train)
    test_score = model.score(X_test, y_test)
    
    print(f"Train accuracy: {train_score:.3f}")
    print(f"Test accuracy: {test_score:.3f}")
    
    # Cross-validation
    cv_scores = cross_val_score(model, X_train, y_train, cv=5)
    print(f"Cross-validation accuracy: {cv_scores.mean():.3f} (+/- {cv_scores.std() * 2:.3f})")
    
    # Classification report
    y_pred = model.predict(X_test)
    print("\nClassification Report:")
    print(classification_report(y_test, y_pred))
    
    print("\nConfusion Matrix:")
    print(confusion_matrix(y_test, y_pred))
    
    # Feature importance
    feature_importance = pd.DataFrame({
        'feature': FEATURE_NAMES,
        'importance': model.feature_importances_
    }).sort_values('importance', ascending=False)
    
    print("\nTop 10 Important Features:")
    print(feature_importance.head(10))
    
    # Export to ONNX
    print(f"\nExporting model to {output_path}...")
    initial_type = [('input', FloatTensorType([None, len(FEATURE_NAMES)]))]
    onnx_model = convert_sklearn(
        model, 
        initial_types=initial_type, 
        target_opset=12,
        options={id(model): {'zipmap': False}}  # Disable zipmap for cleaner output
    )
    
    with open(output_path, 'wb') as f:
        f.write(onnx_model.SerializeToString())
    
    print("Model exported successfully!")
    
    return model

def main():
    parser = argparse.ArgumentParser(description='Train regime classification model')
    parser.add_argument('--clickhouse-url', default='localhost:9000', help='ClickHouse URL (host:port)')
    parser.add_argument('--start-date', default='2023-01-01', help='Training data start date')
    parser.add_argument('--end-date', default='2024-12-31', help='Training data end date')
    parser.add_argument('--output', default='../../models/regime_detector.onnx', help='Output ONNX model path')
    
    args = parser.parse_args()
    
    # Load data
    df = load_training_data(args.clickhouse_url, args.start_date, args.end_date)
    
    if len(df) < 100:
        print("ERROR: Not enough labeled data for training (need at least 100 samples)")
        print("Please label more data using label_data.py script")
        return
    
    # Train and export
    train_model(df, args.output)

if __name__ == '__main__':
    main()


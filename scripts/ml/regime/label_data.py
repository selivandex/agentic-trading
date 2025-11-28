#!/usr/bin/env python3
"""
Interactive script to label historical regime data for training.

Usage:
    python label_data.py --clickhouse-url localhost:9000 --symbol BTC-USDT
"""

import argparse
import pandas as pd
import clickhouse_connect
from datetime import datetime, timedelta

REGIME_TYPES = {
    '1': 'trend_up',
    '2': 'trend_down',
    '3': 'range',
    '4': 'breakout',
    '5': 'volatile',
    '6': 'accumulation',
    '7': 'distribution'
}

def label_data_interactive(clickhouse_url, symbol, start_date):
    """Interactive labeling of regime data."""
    host, port = clickhouse_url.split(':')
    client = clickhouse_connect.get_client(host=host, port=int(port))
    
    # Fetch unlabeled data
    query = f"""
    SELECT timestamp, atr_14, adx, bb_width, volume_change_pct, ema_alignment
    FROM regime_features
    WHERE symbol = '{symbol}'
      AND regime_label = ''
      AND timestamp >= '{start_date}'
    ORDER BY timestamp
    LIMIT 100
    """
    
    result = client.query(query)
    df = pd.DataFrame(result.result_rows, columns=[
        'timestamp', 'atr_14', 'adx', 'bb_width', 'volume_change_pct', 'ema_alignment'
    ])
    
    print(f"\nFound {len(df)} unlabeled samples for {symbol}")
    
    if len(df) == 0:
        print("No unlabeled data found. Either all data is labeled or no features extracted yet.")
        print("Run feature_extractor worker first to generate features.")
        return
    
    labeled_count = 0
    
    for idx, row in df.iterrows():
        print(f"\n{'='*60}")
        print(f"Sample {idx + 1}/{len(df)}")
        print(f"{'='*60}")
        print(f"Timestamp: {row['timestamp']}")
        print(f"ATR (14):        {row['atr_14']:.2f}")
        print(f"ADX:             {row['adx']:.2f}")
        print(f"BB Width:        {row['bb_width']:.2f}%")
        print(f"Volume Change:   {row['volume_change_pct']:.2f}%")
        print(f"EMA Alignment:   {row['ema_alignment']}")
        
        print(f"\n{'Regime Types:':<20}")
        for key, value in REGIME_TYPES.items():
            print(f"  {key}: {value}")
        print(f"  s: Skip")
        print(f"  q: Quit")
        
        label_input = input("\nEnter regime label: ").strip()
        
        if label_input == 'q':
            break
        elif label_input == 's':
            continue
        elif label_input not in REGIME_TYPES:
            print("Invalid input! Skipping...")
            continue
        
        regime_label = REGIME_TYPES[label_input]
        
        # Update database
        update_query = f"""
        ALTER TABLE regime_features
        UPDATE regime_label = '{regime_label}'
        WHERE symbol = '{symbol}' AND timestamp = '{row['timestamp']}'
        """
        
        try:
            client.command(update_query)
            labeled_count += 1
            print(f"✓ Labeled as: {regime_label}")
        except Exception as e:
            print(f"✗ Error labeling: {e}")
    
    print(f"\n{'='*60}")
    print(f"Labeled {labeled_count} samples")
    print(f"{'='*60}")

def main():
    parser = argparse.ArgumentParser(description='Label regime data for training')
    parser.add_argument('--clickhouse-url', default='localhost:9000', help='ClickHouse URL (host:port)')
    parser.add_argument('--symbol', default='BTC-USDT', help='Symbol to label')
    parser.add_argument('--start-date', 
                        default=(datetime.now() - timedelta(days=90)).strftime('%Y-%m-%d'), 
                        help='Start date (YYYY-MM-DD)')
    
    args = parser.parse_args()
    
    print(f"\nStarting interactive regime labeling for {args.symbol}")
    print(f"Date range: from {args.start_date}")
    
    label_data_interactive(args.clickhouse_url, args.symbol, args.start_date)

if __name__ == '__main__':
    main()


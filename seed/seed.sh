#!/usr/bin/env bash
# Seeds a demo "regional_sales" dataset with 3 quarterly versions.
# Requires the backend to be running at http://localhost:8080.
set -e

BASE="http://localhost:8080"

echo "Creating dataset..."
DATASET=$(curl -sf -X POST "$BASE/datasets" \
  -H "Content-Type: application/json" \
  -d '{"name":"regional_sales","primary_key_col":"id"}')
DATASET_ID=$(echo "$DATASET" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Dataset ID: $DATASET_ID"

echo "Uploading Q1 snapshot..."
curl -sf -X POST "$BASE/datasets/$DATASET_ID/snapshots" \
  -F "file=@$(dirname "$0")/v1_sales_q1.csv" \
  -F "message=Q1 baseline: 10 regions × products" > /dev/null
echo "Q1 done"

echo "Uploading Q2 snapshot..."
curl -sf -X POST "$BASE/datasets/$DATASET_ID/snapshots" \
  -F "file=@$(dirname "$0")/v2_sales_q2.csv" \
  -F "message=Q2: added Widget E, removed id=5, revenue up across board" > /dev/null
echo "Q2 done"

echo "Uploading Q3 snapshot..."
curl -sf -X POST "$BASE/datasets/$DATASET_ID/snapshots" \
  -F "file=@$(dirname "$0")/v3_sales_q3.csv" \
  -F "message=Q3: added Widget F (ids 13-14), margins improved, id=8 revenue down" > /dev/null
echo "Q3 done"

echo ""
echo "Seed complete! Open http://localhost:5173 and click regional_sales."
echo "Dataset ID: $DATASET_ID"

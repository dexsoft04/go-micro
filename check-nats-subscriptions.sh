#!/bin/bash
# check-nats-subscriptions.sh

echo "Checking NATS subscriptions..."

# Get all subscriptions
curl -s "http://127.0.0.1:8222/subsz?subs=1" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Total subscriptions: {data[\"num_subscriptions\"]}')
print('\\nSubscriptions for user.item.change:')
for sub in data.get('subscriptions', []):
    if 'user.item.change' in sub.get('subject', ''):
        print(f'  - Queue: {sub.get(\"qgroup\", \"none\")}, Messages: {sub.get(\"msgs\", 0)}, CID: {sub.get(\"cid\", \"?\")}, SID: {sub.get(\"sid\", \"?\")}')

print('\\nAll mcbeam-game-center-srv subscriptions:')
for sub in data.get('subscriptions', []):
    if 'mcbeam-game-center-srv' in sub.get('qgroup', ''):
        print(f'  - Subject: {sub.get(\"subject\", \"unknown\")}, Messages: {sub.get(\"msgs\", 0)}, CID: {sub.get(\"cid\", \"?\")}')
"

# Get connection info
echo -e "\nActive connections:"
curl -s "http://127.0.0.1:8222/connz" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Total connections: {data[\"num_connections\"]}')
for conn in data.get('connections', [])[:10]:
    name = conn.get('name', 'unnamed')
    if 'game-center' in name.lower():
        print(f'  - {name} (CID: {conn.get(\"cid\", \"?\")}): {conn.get(\"subscriptions\", 0)} subscriptions')
"
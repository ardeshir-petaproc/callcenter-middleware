#!/usr/bin/env bash
# End-to-end test against a real Asterisk 18 in Docker, with a fake OpenAI
# Realtime server. Scenarios: queue (IVR, DTMF, ACD, recording, hold, whisper),
# voicemail, outbound click-to-call, CSAT survey, AI agent handoff over
# ExternalMedia RTP, the campaign dialer, and warm transfer / conference.
set -euo pipefail
cd "$(dirname "$0")"
ROOT=$(cd ../.. && pwd)
WORK=$(mktemp -d)
trap 'kill ${MW_PID:-} ${RT_PID:-} 2>/dev/null || true; docker rm -f cc-e2e-asterisk >/dev/null 2>&1 || true; rm -rf "$WORK"' EXIT

cp "$ROOT"/deploy/asterisk/*.conf "$WORK"/
cat asterisk/extensions-test.conf >> "$WORK/extensions.conf"
cat asterisk/ari-test.conf >> "$WORK/ari.conf"
docker run -d --name cc-e2e-asterisk -p 18089:8088 -v "$WORK":/etc/asterisk/custom:ro andrius/asterisk:18 \
  sh -c "cp -f /etc/asterisk/custom/*.conf /etc/asterisk/ && mkdir -p /var/spool/asterisk/recording && chmod 777 /var/spool/asterisk/recording && exec asterisk -f" >/dev/null
GATEWAY=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}')

go build -o "$WORK/fakert" ./fakert && go build -o "$WORK/driver" ./driver
(cd "$ROOT" && go build -o "$WORK/middleware" ./cmd/middleware)
"$WORK/fakert" > "$WORK/fakert.log" 2>&1 & RT_PID=$!
for i in $(seq 30); do curl -sf -u tester:tester http://127.0.0.1:18089/ari/asterisk/info >/dev/null && break; sleep 1; done

env SEED_DEMO=true DB_PATH="$WORK/e2e.db" HTTP_PORT=18080 \
  ARI_URL=http://127.0.0.1:18089/ari ARI_WS_URL=ws://127.0.0.1:18089/ari/events \
  OPENAI_API_KEY=dummy OPENAI_REALTIME_URL=ws://127.0.0.1:18099/v1/realtime OPENAI_BASE_URL=http://127.0.0.1:1/v1 \
  MEDIA_ADVERTISE_HOST="$GATEWAY" DEFAULT_WRAPUP_SEC=0 'OUTBOUND_DIAL_TEMPLATE=Local/{number}@test-customers' \
  "$WORK/middleware" > "$WORK/middleware.log" 2>&1 & MW_PID=$!
sleep 3
for s in queue voicemail outbound survey ai campaign consult; do
  echo "### $s"
  "$WORK/driver" "$s"
  sleep 2
done
echo "All Asterisk e2e scenarios passed."

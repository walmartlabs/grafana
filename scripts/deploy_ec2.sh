#!/bin/bash
ssh ubuntu@54.80.139.44 << EOF
cd go/src/github.com/grafana/grafana
git pull
make
sudo service grafana restart
EOF
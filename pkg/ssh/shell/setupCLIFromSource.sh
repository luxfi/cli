#!/usr/bin/env bash
export PATH=$PATH:~/go/bin
cd ~
rm -rf cli
git clone --single-branch -b {{ .CliBranch }} https://github.com/luxdefi/cli 
cd cli
./scripts/build.sh
cp bin/lux ~/bin/lux

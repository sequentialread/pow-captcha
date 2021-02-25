#!/bin/bash -e

if [ ! -f build_wasm.sh ]; then
  printf "Please run this script from the wasm_build folder.\n"
fi

if [ ! -d scrypt-wasm ]; then
  printf "Cloning https://github.com/MyEtherWallet/scrypt-wasm... \n"
  git clone https://github.com/MyEtherWallet/scrypt-wasm
fi

cd scrypt-wasm

rust_is_installed="$(which rustc | wc -l)"

if [ "$rust_is_installed" == "0" ]; then
  printf "rust language compilers & tools will need to be installed."
  printf "using rustup.rs: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh \n"
  read -p "is this ok? [y] " -n 1 -r
  printf "\n"
  if [[ $REPLY =~ ^[Yy]$ ]]
  then
      curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
  fi
fi

if [ ! -d pkg ]; then
  printf "running Makefile for MyEtherWallet/scrypt-wasm... \n"
  make
fi

cd ../

nodejs_is_installed="$(which node | wc -l)"
npm_is_installed="$(which npm | wc -l)"

if [ "$nodejs_is_installed" == "0" ] || [ "$npm_is_installed" == "0"  ]; then
  printf "nodejs and npm are required for the next step. Please install them manually 😇"
fi

if [ ! -d node_modules ]; then
  printf "running npm install \n"
  npm install
fi

node build_wasm_webworker.js > "../static/proofOfWorkerWASM.js"

printf "\n\nbuilt ../static/proofOfWorkerWASM.js successfully!\n\n"


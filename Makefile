build-lotus:
	git submodule update --init --recursive
	cd ./extern/lotus/extern/filecoin-ffi && make clean && make all
	cd ./extern/lotus && make



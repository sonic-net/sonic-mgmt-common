all: devkit
#change below location to sysroot
DEVKIT_PATH=../platform-root/opt/ngos/inc
HEADERS=$(wildcard inc/*.h)

devkit: $(HEADERS)
	cp $^ $(DEVKIT_PATH)

clean:
	        $(MAKE) -C src clean

.PHONY: all clean src devkit



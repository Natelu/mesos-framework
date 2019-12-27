 #!/bin/bash
cd "$(dirname "$0")"
path=`pwd`
export GOPATH=$(readlink -f $path/../../)

#if [ ! -f "./mesos/mesos.pb.go" ];then
#  protoc --gogo_out=./ --proto_path=./ -I./vendor -I/usr/local/protoc/include ./mesos/mesos.proto
#fi

CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w -extldflags "-static"' main/luzx-framework.go

CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w -extldflags "-static"' main/luzx-executor.go

docker images|grep cke-executor|awk '{print $3}'|xargs docker rmi
docker build -t luzx-executor:0.0.1 .
docker tag luzx-executor:0.0.1 reg.mg.hcbss/open/luzx-executor:0.0.1
docker push reg.mg.hcbss/open/luzx-executor:0.0.1
#CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' #-o bin/welkin-storage $path/welkin-storage.go
p4config = ../config/config.yml
p4config-topo = ../config/topology.json
singlesw-config = ../config/singlesw-config.yml
singlesw-topo = ../config/singlesw-topo.json
3sw-config = ../config/3sw-config.yml
3sw-topo = ../config/3sw-topology.json

all: singlesw
singlesw:
	go run cmd/main.go -n 1 -config $(singlesw-config) -topology $(singlesw-topo) -verbose
3sw:
	go run cmd/main.go -n 2 -config $(3sw-config) -topology $(3sw-topo) -verbose
debug:
	go run cmd/main.go -n 4 -config $(p4config) -topology $(p4config-topo) -verbose
trace:
	go run cmd/main.go -n 4 -config $(p4config) -topology $(p4config-topo) -trace
prod:
	go run cmd/main.go -n 4 -config $(p4config) -topology $(p4config-topo)
tls:
	go run cmd/main.go -n 4 -config $(p4config) -topology $(p4config-topo) -cert-file /tmp/cert.pem -verbose


build:
	DEBUG=true GOPROXY=direct GOSUMDB=off go mod tidy
	DEBUG=true GOPROXY=direct GOSUMDB=off go build -ldflags "-w -s" -buildvcs=false netscaler/kubectl-netscaler.go
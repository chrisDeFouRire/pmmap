all: test docker

test:
	go test

pmmap.docker: *.go
	env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.ListenAddress=:8080" -o pmmap.docker

docker: pmmap.docker
	docker build -t tlsproxy/pmmap .
	docker push tlsproxy/pmmap

clean:
	rm pmmap.docker
	docker images | grep '<none>' | awk '{ print $3 }' | xargs docker rmi
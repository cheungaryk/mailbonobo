# TODO: once SES method is out of alpha, uncomment the run command below
# run:
# 	go run main.go

gen-file:
	go run main.go --saveFile

test:
	go test ./... -cover
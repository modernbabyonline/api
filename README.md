# api - https://api.modernbaby.online

[![Go Report](https://goreportcard.com/badge/github.com/modernbabyonline/api)](https://goreportcard.com/report/github.com/modernbabyonline/api) [![GolangCI](https://golangci.com/badges/github.com/modernbabyonline/api.svg)](https://golangci.com/r/github.com/modernbabyonline/api) [![Travis](https://travis-ci.org/modernbabyonline/api.svg?branch=master)](https://travis-ci.org/modernbabyonline/api#) [![codecov](https://codecov.io/gh/modernbabyonline/api/branch/master/graph/badge.svg)](https://codecov.io/gh/modernbabyonline/api) ![License](https://img.shields.io/github/license/modernbabyonline/api.svg) ![Maintenance](https://img.shields.io/maintenance/yes/2018.svg) [![GoDoc](https://godoc.org/github.com/modernbabyonline/api?status.svg)](https://godoc.org/github.com/modernbabyonline/api)

## Requirements:
- go - `brew install go`
- dep - `brew install dep`
- docker - `brew install docker`
- make - `brew install make`

## Commands:
```bash
make test
make build
make seedme
```

- `make test`: runs go tests in verbose mode
- `make build`: runs the docker-compose to spin up the project
- `make seedme`: seeds datbase - run in different terminal window while running `make build`

## Services:
- api server on `localhost:8000`
- mongodb server on `localhost:27017`
- mongodb express server on `localhost:8081`

## Browse MongoDB Database:
- http://localhost:8081

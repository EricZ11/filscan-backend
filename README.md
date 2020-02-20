# Overview

Filscan is a blockchain browser for Filecoin, which can be used to view Filecoin blockchain data, including querying addresses, messages information, block heights, miner information, token information, etc.

# Table of Contents
- [Overview](#overview)
- [Table of Contents](#table-of-contents)
- [Front-End](#front-end)
- [Back-End](#back-end)
  - [Build and Install](#build-and-install)
    - [Environment](#environment)
    - [System Require](#system-require)
    - [Build](#build)
    - [Configuration](#configuration)
    - [Run](#run)
  - [API Document](#api-document)

# [Front-End](https://github.com/ipfs-force-community/filscan-frontend)


# Back-End

## Build and Install

### Environment

- golang >= v1.13
- mongo >= v4.2
- lotus >= v0.2.7

### System Require

- Linux or Mac OS

### Build
```
git clone (githuburl)

cd Backend

make build-lotus

go build
```
### Configuration

Edit app.conf in path /conf and set the correct parameter
```
mongoHost = "127.0.0.1:27017"

mongoUser = "root"

mongoPass = "admin"

mongoDB   = "filscan"

lotusGetWay="192.168.1.1:1234"
```
### Run

Make sure mongo and lotus is active, and run the filscan_lotus
```
./filscan_lotus
```
The application will check lotus and mongo’s status. The application will stop if got any error from them. If application start success, it will work until sync all data down from lotus. 

## API Document

Check document [here](Filscan_Interface_v1.0.md)

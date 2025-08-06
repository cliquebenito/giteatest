# Directories
BIN_DIR = bin
CUSTOM_HOOKS_DIR = bin/custom_hooks
CMD_DIR = cmd

## pre-receive directories
PRE_RECEIVE = pre-receive
PRE_RECEIVE_DIR = ${CUSTOM_HOOKS_DIR}/${PRE_RECEIVE}.d

## update directories
UPDATE = update
UPDATE_DIR = ${CUSTOM_HOOKS_DIR}/${UPDATE}.d

## proc-receive directories
PROC_RECEIVE = proc-receive
PROC_RECEIVE_DIR = ${CUSTOM_HOOKS_DIR}/${PROC_RECEIVE}.d

## post-receive directories
POST_RECEIVE = post-receive
POST_RECEIVE_DIR = ${CUSTOM_HOOKS_DIR}/${POST_RECEIVE}.d

.PHONY:	all
all:	clean create-dirs build

.PHONY:	all-linux
all-linux:	clean create-dirs build-linux

.PHONY:	build
build:	pre-receive	update proc-receive post-receive

.PHONY:	build-linux
build-linux:	pre-receive-linux update-linux proc-receive-linux post-receive-linux

.PHONY:	pre-receive
pre-receive:
	go build -o ${PRE_RECEIVE_DIR}/${PRE_RECEIVE} ${CMD_DIR}/${PRE_RECEIVE}/main.go

.PHONY:	update
update:
	go build -o ${UPDATE_DIR}/${UPDATE} ${CMD_DIR}/${UPDATE}/main.go

.PHONY:	proc-receive
proc-receive:
	go build -o ${PROC_RECEIVE_DIR}/${PROC_RECEIVE} ${CMD_DIR}/${PROC_RECEIVE}/main.go

.PHONY:	post-receive
post-receive:
	go build -o ${POST_RECEIVE_DIR}/${POST_RECEIVE} ${CMD_DIR}/${POST_RECEIVE}/main.go

.PHONY:	pre-receive-linux
pre-receive-linux:
	env GOOS=linux GOARCH=amd64 go build -o ${PRE_RECEIVE_DIR}/${PRE_RECEIVE} ${CMD_DIR}/${PRE_RECEIVE}/main.go

.PHONY:	update-linux
update-linux:
	env GOOS=linux GOARCH=amd64 go build -o ${UPDATE_DIR}/${UPDATE} ${CMD_DIR}/${UPDATE}/main.go

.PHONY:	proc-receive-linux
proc-receive-linux:
	env GOOS=linux GOARCH=amd64 go build -o ${PROC_RECEIVE_DIR}/${PROC_RECEIVE} ${CMD_DIR}/${PROC_RECEIVE}/main.go

.PHONY:	post-receive-linux
post-receive-linux:
	env GOOS=linux GOARCH=amd64 go build -o ${POST_RECEIVE_DIR}/${POST_RECEIVE} ${CMD_DIR}/${POST_RECEIVE}/main.go


.PHONY:	create-dirs
create-dirs:
	mkdir ${BIN_DIR} \
		&& mkdir ${CUSTOM_HOOKS_DIR} \
		&& mkdir ${PRE_RECEIVE_DIR} \
		&& mkdir ${UPDATE_DIR} \
		&& mkdir ${PROC_RECEIVE_DIR} \
		&& mkdir ${POST_RECEIVE_DIR}

.PHONY:	clean
clean:
	 rm -rf ${BIN_DIR}
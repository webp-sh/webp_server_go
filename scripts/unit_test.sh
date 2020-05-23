#!/usr/bin/env bash

# bash scripts/unit_test.sh
# check $? for success or failure
go test -v -cover encoder_test.go encoder.go helper.go
go test -v -cover helper_test.go helper.go

# if [[ $? -ne 0 ]] ; then
#  echo "TEST FAILED!!! PLEASE DOUBLE CHECK."
# fi

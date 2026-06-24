#!/usr/bin/env bash
# Test procedure: echo args then stdin
read -r sin
echo "$@"
echo "$sin"

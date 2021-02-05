#!/bin/sh
set -e -x

# Verify our environment variables are set
[ -z "${COMMIT_USER}" ] && { echo "Need to set COMMIT_USER"; exit 1; }
[ -z "${COMMIT_EMAIL}" ] && { echo "Need to set COMMIT_EMAIL"; exit 1; }
[ -z "${SSH_KEY}" ] && { echo "Need to set SSH_KEY"; exit 1; }

# Set up our SSH Key
if [ ! -d ~/.ssh ]; then
	echo "SSH Key was not found. Configuring SSH Key."
	mkdir ~/.ssh
	echo -e "${SSH_KEY}" > ~/.ssh/id_rsa
	chmod 700 ~/.ssh
	chmod 600 ~/.ssh/id_rsa

	echo -e "Host *\n    StrictHostKeyChecking no\n    UserKnownHostsFile=/dev/null\n" > ~/.ssh/config
fi

# Configure our user and email to commit as.
git config --global user.name "${COMMIT_USER}"
git config --global user.email "${COMMIT_EMAIL}"

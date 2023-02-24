#SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
#
#  SPDX-License-Identifier: Apache-2.0

IMAGE_NAME := "eu.gcr.io/gardener-project/certificate-dns-bridge"
IMAGE_TAG := "latest"

build:
	docker build --platform=linux/amd64 -t "$(IMAGE_NAME):$(IMAGE_TAG)" .
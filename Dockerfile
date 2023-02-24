#SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
#
#  SPDX-License-Identifier: Apache-2.0

FROM golang:1.19-alpine AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace
ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /

COPY --from=build /workspace/webhook /webhook

ENTRYPOINT ["/webhook"]

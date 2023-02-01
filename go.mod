module github.com/aperturerobotics/util

go 1.19

replace (
	github.com/sirupsen/logrus => github.com/aperturerobotics/logrus v1.9.1-0.20221224130652-ff61cbb763af // aperture
	google.golang.org/protobuf => github.com/aperturerobotics/protobuf-go v1.28.2-0.20230110194655-55a09796292e // aperture
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fsnotify/fsnotify v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/google/go-cmp v0.5.7 // indirect
	golang.org/x/sys v0.4.1-0.20230131160137-e7d7f63158de // indirect
)

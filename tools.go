// +build tools

package main

import (
	_ "github.com/git-chglog/git-chglog/cmd/git-chglog"
	_ "github.com/google/addlicense"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/mcubik/goverreport"
	_ "github.com/psampaz/go-mod-outdated"
	_ "github.com/securego/gosec/cmd/gosec"
	_ "github.com/segmentio/golines"
	_ "github.com/spf13/cobra/cobra"
	_ "github.com/uw-labs/lichen"
	_ "mvdan.cc/gofumpt"
)

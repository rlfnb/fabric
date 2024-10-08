/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channel

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric/internal/peer/common"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func fetchCmd(cf *ChannelCmdFactory) *cobra.Command {
	fetchCmd := &cobra.Command{
		Use:   "fetch <newest|oldest|config|(number)> [outputfile]",
		Short: "Fetch a block",
		Long:  "Fetch a specified block, writing it to a file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetch(cmd, args, cf)
		},
	}
	flagList := []string{
		"channelID",
		"bestEffort",
	}
	attachFlags(fetchCmd, flagList)

	return fetchCmd
}

func fetch(cmd *cobra.Command, args []string, cf *ChannelCmdFactory) error {
	if len(args) == 0 {
		return fmt.Errorf("fetch target required, oldest, newest, config, or a number")
	}
	if len(args) > 2 {
		return fmt.Errorf("trailing args detected")
	}
	// Parsing of the command line is done so silence cmd usage
	cmd.SilenceUsage = true

	// default to fetching from orderer
	ordererRequired := OrdererRequired
	peerDeliverRequired := PeerDeliverNotRequired
	if len(strings.Split(common.OrderingEndpoint, ":")) != 2 {
		// if no orderer endpoint supplied, connect to peer's deliver service
		ordererRequired = OrdererNotRequired
		peerDeliverRequired = PeerDeliverRequired
	}
	var err error
	if cf == nil {
		cf, err = InitCmdFactory(EndorserNotRequired, peerDeliverRequired, ordererRequired)
		if err != nil {
			return err
		}
	}

	var block *cb.Block

	switch args[0] {
	case "oldest":
		block, err = cf.DeliverClient.GetOldestBlock()
	case "newest":
		block, err = cf.DeliverClient.GetNewestBlock()
	case "config":
		iBlock, err2 := cf.DeliverClient.GetNewestBlock()
		if err2 != nil {
			return err2
		}
		lc, err2 := protoutil.GetLastConfigIndexFromBlock(iBlock)
		if err2 != nil {
			return err2
		}
		logger.Infof("Retrieving last config block: %d", lc)
		block, err = cf.DeliverClient.GetSpecifiedBlock(lc)
	default:
		num, err2 := strconv.Atoi(args[0])
		if err2 != nil {
			return fmt.Errorf("fetch target illegal: %s", args[0])
		}
		block, err = cf.DeliverClient.GetSpecifiedBlock(uint64(num))
	}
	if err != nil {
		return err
	}

	if block == nil {
		return errors.New("proto: Marshal called with nil")
	}
	b, err := proto.Marshal(block)
	if err != nil {
		return err
	}

	var file string
	if len(args) == 1 {
		file = channelID + "_" + args[0] + ".block"
	} else {
		file = args[1]
	}

	if err = os.WriteFile(file, b, 0o644); err != nil {
		return err
	}

	return nil
}

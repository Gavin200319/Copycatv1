package copycatv1

import (
	"bufio"

	startReceiver "copycatv1/app/copycat-v1/modes/lan/startReceiver"
	startListener "copycatv1/app/copycat-v1/modes/lan/startListener"
	startBroadcaster "copycatv1/app/copycat-v1/modes/lan/startBroadcaster"
	menuLoop "copycatv1/app/copycat-v1/menuLoop"
)

func StartNetworkMode(reader *bufio.Reader) {
	go startReceiver.StartReceiver()
	go startListener.StartListener()
	go startBroadcaster.StartBroadcaster()

	menuLoop.MenuLoop(reader)
}

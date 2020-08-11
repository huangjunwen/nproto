package sendmail

var (
	// SvcSpec is the service spec for rpc server/client.
	SvcSpec = NewSendMailSvcSpec("sendmail")
)

var (
	// QueueSpec is the msg spec for publisher/subscriber.
	QueueSpec = EmailEntrySpec("sendmail.queue")
)

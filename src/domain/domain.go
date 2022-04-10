package domain

type ScrambleStreams struct {
	Streams map[string]Stream
}

type TagOptions struct {
	Name       string
	MultiValue string
	SubValue   string
	Pattern    string
}

type Stream struct {
	Tables []string
	Tags   []TagOptions
}

type T24FieldMS struct {
	Text string `xml:",chardata"`
	M    string `xml:"m,attr"`
	S    string `xml:"s,attr"`
}

type T24FieldM struct {
	Text string `xml:",chardata"`
	M    string `xml:"m,attr"`
}

type T24Field struct {
	Text string `xml:",chardata"`
}

type TagKey struct {
	Name       string
	MultiValue string
	SubValue   string
}

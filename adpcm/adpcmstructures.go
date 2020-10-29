package adpcm

const PREDICTOR_SIZE = 8

type Frame struct {
	Header uint8
	Data   [8]uint8
}

type Predictor struct {
	Table [PREDICTOR_SIZE][]int32
}

type Codebook struct {
	Predictors []Predictor
	Order      int
}

type Loop struct {
	Start int
	End   int
	Count int
	State [16]int16
}

type ADPCMEncodedData struct {
	NSamples   int
	SampleRate float64
	Codebook   *Codebook
	Loop       *Loop
	Frames     []Frame
}

type PCMEncodedData struct {
	Samples []int16
}

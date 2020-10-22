package adpcm

const PREDICTOR_SIZE = 8

type Frame struct {
	Header uint8
	Data   [8]uint8
}

type Predictor struct {
	Table [PREDICTOR_SIZE][]int32
	Order int
}

type Codebook struct {
	Predictors []Predictor
}

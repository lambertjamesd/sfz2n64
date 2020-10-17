package al64

import (
	"encoding/binary"
	"io"
)

type alSerializable interface {
	serializeWrite(state *alSerializeState, target io.Writer)
	sizeInBytes() int
	generateLayout(state *alSerializeState)
}

type alSerializeState struct {
	offsetMapping   map[alSerializable]int
	pending         []alSerializable
	currentLocation int
}

func (state *alSerializeState) layoutSerializable(target alSerializable) {
	_, exists := state.offsetMapping[target]

	if !exists {
		state.offsetMapping[target] = state.currentLocation
		state.currentLocation = state.currentLocation + target.sizeInBytes()
		state.pending = append(state.pending, target)

		target.generateLayout(state)
	}
}

func (state *alSerializeState) getSerializableOffset(target alSerializable) int32 {
	result, exists := state.offsetMapping[target]

	if exists {
		return int32(result)
	} else {
		return 0
	}
}

func (state *alSerializeState) writeOut(target io.Writer) {
	for _, serializable := range state.pending {
		serializable.serializeWrite(state, target)
	}
}

// ALADPCMBook
func (book *ALADPCMBook) serializeWrite(state *alSerializeState, target io.Writer) {

}

func (book *ALADPCMBook) sizeInBytes() int {
	return 8
}

func (book *ALADPCMBook) generateLayout(state *alSerializeState) {

}

// ALADPCMloop
func (loop *ALADPCMloop) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &loop.Start)
	binary.Write(target, binary.BigEndian, &loop.End)
	binary.Write(target, binary.BigEndian, &loop.Count)

	for i := 0; i < ADPCMFSIZE; i = i + 1 {
		binary.Write(target, binary.BigEndian, &loop.State[i])
	}
}

func (loop *ALADPCMloop) sizeInBytes() int {
	return 44
}

func (loop *ALADPCMloop) generateLayout(state *alSerializeState) {

}

// ALRawLoop
func (rawLoop *ALRawLoop) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &rawLoop.Start)
	binary.Write(target, binary.BigEndian, &rawLoop.End)
	binary.Write(target, binary.BigEndian, &rawLoop.Count)
}

func (rawLoop *ALRawLoop) sizeInBytes() int {
	return 12
}

func (rawLoop *ALRawLoop) generateLayout(state *alSerializeState) {

}

// ALWavetable
func (waveTable *ALWavetable) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &waveTable.Base)
	binary.Write(target, binary.BigEndian, &waveTable.Len)
	binary.Write(target, binary.BigEndian, &waveTable.Type)
	var flags uint8 = 0
	binary.Write(target, binary.BigEndian, &flags)
	// padding
	binary.Write(target, binary.BigEndian, &flags)
	binary.Write(target, binary.BigEndian, &flags)

	if waveTable.Type == AL_ADPCM_WAVE {
		var offset = state.getSerializableOffset(waveTable.AdpcWave.Loop)
		binary.Write(target, binary.BigEndian, &offset)
		offset = state.getSerializableOffset(waveTable.AdpcWave.Book)
		binary.Write(target, binary.BigEndian, &offset)
	} else {
		var offset = state.getSerializableOffset(waveTable.RawWave.Loop)
		binary.Write(target, binary.BigEndian, &offset)
	}
}

func (waveTable *ALWavetable) sizeInBytes() int {
	return 20
}

func (waveTable *ALWavetable) generateLayout(state *alSerializeState) {
	if waveTable.Type == AL_ADPCM_WAVE {
		state.layoutSerializable(waveTable.AdpcWave.Loop)
		state.layoutSerializable(waveTable.AdpcWave.Book)
	} else {
		state.layoutSerializable(waveTable.RawWave.Loop)
	}
}

// ALKeyMap
func (keyMap *ALKeyMap) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &keyMap.VelocityMin)
	binary.Write(target, binary.BigEndian, &keyMap.VelocityMax)
	binary.Write(target, binary.BigEndian, &keyMap.KeyMin)
	binary.Write(target, binary.BigEndian, &keyMap.KeyMax)
	binary.Write(target, binary.BigEndian, &keyMap.KeyBase)
	binary.Write(target, binary.BigEndian, &keyMap.Detune)
}

func (keyMap *ALKeyMap) sizeInBytes() int {
	return 6
}

func (keyMap *ALKeyMap) generateLayout(state *alSerializeState) {

}

// ALEnvelope
func (envelope *ALEnvelope) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &envelope.AttackTime)
	binary.Write(target, binary.BigEndian, &envelope.DecayTime)
	binary.Write(target, binary.BigEndian, &envelope.ReleaseTime)
	binary.Write(target, binary.BigEndian, &envelope.AttackVolume)
	binary.Write(target, binary.BigEndian, &envelope.DecayVolume)
}

func (envelope *ALEnvelope) sizeInBytes() int {
	return 16
}

func (envelope *ALEnvelope) generateLayout(state *alSerializeState) {

}

// ALSound
func (sound *ALSound) serializeWrite(state *alSerializeState, target io.Writer) {
	var offset = state.getSerializableOffset(sound.Envelope)
	binary.Write(target, binary.BigEndian, &offset)
	offset = state.getSerializableOffset(sound.KeyMap)
	binary.Write(target, binary.BigEndian, &offset)
	offset = state.getSerializableOffset(sound.Wavetable)
	binary.Write(target, binary.BigEndian, &offset)

	binary.Write(target, binary.BigEndian, &sound.SamplePan)
	binary.Write(target, binary.BigEndian, &sound.SampleVolume)
	var flags uint8 = 0
	binary.Write(target, binary.BigEndian, &flags)
}

func (sound *ALSound) sizeInBytes() int {
	return 16
}

func (sound *ALSound) generateLayout(state *alSerializeState) {
	state.layoutSerializable(sound.Envelope)
	state.layoutSerializable(sound.KeyMap)
	state.layoutSerializable(sound.Wavetable)
}

// ALInstrument
func (inst *ALInstrument) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &inst.Volume)
	binary.Write(target, binary.BigEndian, &inst.Pan)
	binary.Write(target, binary.BigEndian, &inst.Priority)
	var flags uint8 = 0
	binary.Write(target, binary.BigEndian, &flags)

	binary.Write(target, binary.BigEndian, &inst.TremType)
	binary.Write(target, binary.BigEndian, &inst.TremRate)
	binary.Write(target, binary.BigEndian, &inst.TremDepth)
	binary.Write(target, binary.BigEndian, &inst.TremDelay)

	binary.Write(target, binary.BigEndian, &inst.VibType)
	binary.Write(target, binary.BigEndian, &inst.VibRate)
	binary.Write(target, binary.BigEndian, &inst.VibDepth)
	binary.Write(target, binary.BigEndian, &inst.VibDelay)

	binary.Write(target, binary.BigEndian, &inst.BendRange)
	var soundCount int16 = int16(len(inst.SoundArray))
	binary.Write(target, binary.BigEndian, &soundCount)

	for _, sound := range inst.SoundArray {
		var soundOffset = state.getSerializableOffset(sound)
		binary.Write(target, binary.BigEndian, &soundOffset)
	}
}

func (inst *ALInstrument) sizeInBytes() int {
	return 16 + 4*len(inst.SoundArray)
}

func (inst *ALInstrument) generateLayout(state *alSerializeState) {
	for _, sound := range inst.SoundArray {
		state.layoutSerializable(sound)
	}
}

// ALBank
func (bank *ALBank) serializeWrite(state *alSerializeState, target io.Writer) {
	var instcount int16 = int16(len(bank.InstArray))
	binary.Write(target, binary.BigEndian, &instcount)
	var flags uint8 = 0
	binary.Write(target, binary.BigEndian, &flags)
	binary.Write(target, binary.BigEndian, &bank.Pad)
	binary.Write(target, binary.BigEndian, &bank.SampleRate)

	var percussionOffset = state.getSerializableOffset(bank.Percussion)
	binary.Write(target, binary.BigEndian, &percussionOffset)

	for _, instrument := range bank.InstArray {
		var instumentOffset = state.getSerializableOffset(instrument)
		binary.Write(target, binary.BigEndian, &instumentOffset)
	}
}

func (bank *ALBank) sizeInBytes() int {
	return 12 + 4*len(bank.InstArray)
}

func (bank *ALBank) generateLayout(state *alSerializeState) {
	if bank.Percussion != nil {
		state.layoutSerializable(bank.Percussion)
	}

	for _, instrument := range bank.InstArray {
		state.layoutSerializable(instrument)
	}
}

// ALBankFile
func (bankFile *ALBankFile) serializeWrite(state *alSerializeState, target io.Writer) {
	binary.Write(target, binary.BigEndian, &bankFile.Revision)
	var bankCount int16 = int16(len(bankFile.BankArray))
	binary.Write(target, binary.BigEndian, &bankCount)

	for _, bank := range bankFile.BankArray {
		var ouput = state.getSerializableOffset(bank)
		binary.Write(target, binary.BigEndian, &ouput)
	}
}

func (bankFile *ALBankFile) sizeInBytes() int {
	return 4 + 4*len(bankFile.BankArray)
}

func (bankFile *ALBankFile) generateLayout(state *alSerializeState) {
	for _, bank := range bankFile.BankArray {
		state.layoutSerializable(bank)
	}
}

func (bankFile *ALBankFile) Serialize(target io.Writer) {
	var state alSerializeState = alSerializeState{
		make(map[alSerializable]int),
		nil,
		0,
	}

	state.layoutSerializable(bankFile)
	state.writeOut(target)
}

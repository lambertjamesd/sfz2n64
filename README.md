# sfz2n64

Converts SFZ files to a format the N64 can use as part of instrument banks

## Usage

sfz2n64 takes the first parameter as the input, which can be an .sfz file, .ins or a .ctl file. The second parameter is the output which can be a .ins, .ctl, or a .sfz file.

For example

This will take input.sfz and convert it into a ctl instrument bank and a tbl file

```
sfz2n64 input.sfz -o output.ctl
```

Changing the extension of the output changes the format

```
sfz2n64 input.sfz -o output.ins
```

This will also create a sounds folder to to save all of the sound data to

## Making instrument banks with sfz

A single sfz file represents a single instrument wherease a .ins file or a .ctl file
may contain multiple instruments. To solve this problem a custom sfz file, not actually an instrument, is needed for combining multiple sfz instruments into a bank The format is described below

```
// use a bank section to denote the start of a new instrument bank
<bank>

// use the <percussion> tag to specify the percussion instrument for the currennt bank
<percussion>
// instrument indicates the .sfz instrument file relative to the current sfz file where the instrument is located
instrument=./instruments/Percussion.sfz

// <instrument> indicates an instrument to add to the current bank
<instrument>
// specifies the General MIDI instrument associated with the current instrument starting at index 1 going up to 128
program_number=1
// specifies the sfz instrument to associate with the program number
instrument=./instruments/Acoustic_Grand_Piano.sfz

// you can have mulitple instruments for a single bank
<instrument>
// you cannot have duplicate program_number values for a bank
program_number=5
instrument=./instruments/Bright_Acoustic_Piano.sfz
```

## --bank_sequence_mapping

This flag can be used to filter unused instruments and sounds out of an instrument bank based on a list of midi files that use the the instrument bank. So for example, suppose
you had a project with an instrument bank file, instruments.ins and a list of all the midi
files that use that instrument bank. You could write a file to specify which midi files use which banks inside that bank file

### song_mapping.list
```
0
song0.mid
song1.mid
1
song2.mid
```
The above file specifies that song0.mid and song1.mid are mapped to bank 0 in the instrument bank instruments.ins and song2.mid is mapped to bank 1. Running the following
command

`sfz2n64 -o instruments.ctl instruments.ins --bank_sequence_mapping song_mapping.list`

Will convert the .ins file into a .ctl file and filter out any unused sounds and 
instruments based on the song_mapping.list file.

## Sound bank extraction

sfz2n64 can also be used to extract sound banks from roms.

`sfz2n64 -o game_sounds.sfz game.n64`

Keep in mind this may not work with all roms if they use another format for sound banks

## Compressing audio

sfz2n64 can also be used to compress audio clips

`sfz2n64 -o clip.aifc clip.aiff`

You can also include the `--compress` flag when converting sound banks to compress any
uncompressed audio.

`sfz2n64 -o instruments.ctl instruments.ins --compress`

When compressing any audio, the following flags will effect compression behavior

| Flag | Description | Default Value | Min Value | Max Value |
| :--- | :---------- | :------------ | :-------- | :-------- |
| --order | the order used in adpcm compression | 2 | 1 | 16 |
--threshold | silence threshold | 10 | 1 | 32 |
--bits | the number of bits to use for adpcm compression | 2 | 1 | 4 |
--refine-iterations | the number of refinement iterations to use in adpcm compression | 2 | 1 | 20000 |


## Additional features

sfz2n64 can work as a replacement to ic.exe. Here are some features added to this tool.

### wav files

Sounds in instruments banks can reference .wav files on top of .aiff files

```
sound Sound {
    use("AstronautFootsteps.wav")
    pan=64
    envelope = AstronautFootstepsEnvelope
}
```

### loopStart, loopEnd, loopCount

Sounds can specify the loopStart, loopEnd, and loopCount values inside an .ins file instead of
having to embed them into an aiff file. 

```
sound Sound {
    use("AstronautFootsteps.wav")
    loopCount = -1
    loopEnd = -1
    pan=64
    envelope = AstronautFootstepsEnvelope
}
```

loopEnd and loopStart are measured in the number of samples from the beginning of the
sound file. If they are negative, they count backwards from the end of the sound file.

The above example loops the entire sound effect from beginning to end indefinitely.

### optional semicolons

Semicolons are optional in sfz2n64
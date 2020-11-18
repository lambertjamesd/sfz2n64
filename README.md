# sfz2n64

Converts SFZ files to a format the N64 can use as part of instrument banks

## Usage

sfz2n64 takes the first parameter as the input, which can be an .sfz file, .ins or a .ctl file. The second parameter is the output which can be a .ins, .ctl, or a .sfz file.

For example

This will take input.sfz and convert it into a ctl instrument bank and a tbl file

```
sfz2n64 input.sfz output.ctl
```

Changing the extension of the output changes the format

```
sfz2n64 input.sfz output.ins
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
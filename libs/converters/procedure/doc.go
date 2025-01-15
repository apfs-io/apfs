/*
Package procedure provides functionality to call of external programs to process
file objects throught STDIN or TMPFILE interfaces.

All subprograms have to implement JSON api interface throught STDOUT pipelines
or pure file response. JSON will be detected automaticly

Example STDOUT file:

	procedure resize-image.sh

Example STOUT json:

	[4]byte{JSON_SIZE_BIGENDING}
	{
		"field1": "value1",
		"field2": "value2",
	}
	[...FILE_STREAM_RESPONSE...]

Example STDOUT tmpfile:

	shell exec magick {{input_tmp_file}} {{output_tmp_file}}
	STDOUT: optional as JSON
*/
package procedure

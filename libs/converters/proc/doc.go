// Package proc implements a [workflow.StepRunner] backed by
// [github.com/demdxx/plugeproc], providing shell, exec, and Docker execution
// modes for APFS v2 workflow steps.
//
// # Overview
//
// Each workflow step declares its execution mode via the uses: field. This
// package handles the following modes:
//
//   - shell   – run an inline bash script from the step's run: field
//   - exec    – run a named procedure loaded from the procedure store
//   - procedure – alias for exec (same behaviour)
//   - docker  – run a command inside a Docker container
//
// Steps with only a docker: block (no uses: field) are also accepted.
//
// # Procedure Store
//
// Named procedures live in the deploy/procedures/ directory. Each procedure
// consists of a shell/Python script and a companion .eproc.yaml manifest that
// declares its parameters and output type. The [Store] is loaded once at
// startup via [NewStore] and passed to [New].
//
// # Step Syntax
//
// Inline shell step:
//
//	steps:
//	  - name: Strip EXIF
//	    uses: shell
//	    with: { target: clean.jpg }
//	    run: |
//	      magick mogrify -strip "{{inputFile}}" -interlace Plane "{{outputFile}}"
//
// Named procedure step (loaded from the store):
//
//	steps:
//	  - name: Resize to 1200 px
//	    uses: procedure
//	    with:
//	      name: image-resize-w
//	      width: "1200"
//	      target: large.jpg
//
// Docker step (image + optional inline script):
//
//	steps:
//	  - name: Transcode with FFmpeg
//	    uses: docker
//	    docker:
//	      image: jrottenberg/ffmpeg:4.4-alpine
//	      remove_after_done: true
//	    with: { target: out.mp4 }
//	    run: |
//	      ffmpeg -i /dev/stdin -vf "scale={{width}}:-1" -f mp4 /dev/stdout
//
// # Parameter Mapping
//
// Values in with: are mapped to the positional parameters declared in the
// .eproc.yaml manifest. The special keys target and input are consumed by the
// runner itself (to route the output path and provide the object's content as
// stdin) and are not forwarded as user parameters.
package proc

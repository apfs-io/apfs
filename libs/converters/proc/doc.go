// Package proc provides a workflow.StepRunner backed by github.com/demdxx/plugeproc.
//
// It replaces the legacy libs/converters/shell and libs/converters/procedure
// packages with a single, unified runner that supports three execution modes:
//
//   - shell   – run an inline bash script from the step's run: block
//   - exec    – run an auto-discovered procedure file from deploy/procedures/
//   - docker  – run a command inside a Docker container
//
// # Procedure Store
//
// Procedures in the deploy/procedures directory are described by .eproc.yaml
// manifests placed alongside their scripts. The Store is loaded once at startup
// via NewStore and passed to the runner.
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
//	      magick mogrify -strip "{{inputFile}}" -interlace Plane -auto-orient "{{outputFile}}"
//
// Named procedure step (loads from store):
//
//	steps:
//	  - name: Resize to 1200 px
//	    uses: procedure
//	    with:
//	      name: image-resize-w
//	      width: "1200"
//
// Docker step:
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
package proc

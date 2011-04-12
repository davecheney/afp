include $(GOROOT)/src/Make.inc

TARG=github.com/davecheney/afp
GOFILES=\
	afp.go\
	buffer.go\
	dsi.go\

include $(GOROOT)/src/Make.pkg

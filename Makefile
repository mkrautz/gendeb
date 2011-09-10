include $(GOROOT)/src/Make.inc

TARG = gendeb

GOFILES =\
	gendeb.go\
	args.go

include $(GOROOT)/src/Make.cmd

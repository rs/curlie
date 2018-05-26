#!/bin/bash

curl -h|perl -pe 's/^\s*(?:-([^-]),\s+)?--(.*?)\s+<.*?>.*/push @a, $1; push @b, $2/e; undef $_; END {print "package args\n\nvar (\n\tcurlShortValues = \"", @a, "\"\n\tcurlLongValues  = []string{", join(", ", map {"\"$_\""} @b), "}\n)\n"}' > curlopts.go

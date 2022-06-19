# gh-edu-plagiarism
Plagiarism is a plug-in for the gh-edu ecosystem to detect plagiarism in programming assignments

> Plagio es la copia servil o imitación torpe de un modelo, con pretensiones de originalidad.\
> Plagiarism is the slavish copying or clumsy imitation of a model, with pretensions of originality.\
> -- <cite>Correa y Lázaro</cite>

## Features
- Highly concurrent: Concurrency and parallelism with no bottle necks so you can get results as fast as possible
- Graceful degradation: If some process fails it tries to give you at least some results (moss URL, report file, graph image)
- Works well in scale: It doesn't matter how many assignments or students are in your organization, plagiarism finds the balance between speed and memory consumption
- Useful for scripting: It uses fzf to ask for user input but is totally functional trough CLI flags.

## How does it work?
Plagiarism depends on the Stanford service MOSS, it clones all the repositories related to an assingment in an organization and sends it to MOSS service. Thereupon, it sends the result to a python script (mossum) that generates a graph.

## Installation
### Dependencies
- [MOSS](https://theory.stanford.edu/~aiken/moss/) script
    - Make sure it can be executed ``chmod ug+x moss`` 
- Perl
- Python 3
- [mossum](https://github.com/hjalti/mossum) script installed
- fzf (optional)

### As a [gh-edu](https://github.com/gh-cli-for-education/gh-edu) plugin:
1. Install as a ``gh-edu`` extension ``gh edu install plagiarism``
2. Move or copy the moss script to the root directory ``mv moss ~/.local/share/gh/extensions/gh-edu-plagiarism``

### Stand-alone:
1. Get the binary 
- Get the binary on releases
- Clone the repository and compile it (You will need go 1.18 or more recent)
- Use go install
```
go install https://github.com/gh-cli-for-education/gh-edu-plagiarism@latest
```
2. Move or copy the moss script to the same directory of the binary

## Compatible languages
- c
- cc (C++)
- java
- ml (Meta Language)
- pascal
- ada
- lisp
- scheme
- haskell
- fortran
- ascii
- vhdl
- perl
- matlab
- python
- mips
- prolog
- spice
- vb (Visual Basic)
- csharp (C#)
- modula2
- a8086 (8086 assembly)
- javascript
- plsql (PL/SQL)
- verilog

It looks like the original creator of MOSS lost the source code and the server
is running on a binary. So it's very unlikely that more languages are added\
https://www.quora.com/Why-is-the-MOSS-measure-of-the-software-similarity-algorithm-not-open-sourced

## Usage
Extracted from the ``--help`` flag
```
Usage:
  gh edu plagiarism [-a] [-q] [-l [<language>]] [-t] [flags]

Flags:
  -a, --anonymize                                                                                 Indicate if you want to randomize the names
  -h, --help                                                                                      help for gh
  -l, --language https://github.com/gh-cli-for-education/gh-edu-plagiarism#compatible-languages   Select the language
  -q, --quiet                                                                                     No INFO in the output only the result
  -t, --template                                                                                  Indicate if there is a tutor template
```
Plagiarism print a file with a report of all the possible pairs in the standard output, just redirect the output if you want to save that information.
```
gh edu plagiarism -q > report.txt
```
It also generate a very useful (and temporary) graph to get an overview, if you want to save it just use the --output flag to indicate the path\
Otherwise it will be in your system temporary directory until the next execution

## Known limitations:
Go uses google's re2 regular expression engine, which have been designed for security and predictable performance
Sadly enough this means that look-arounds are not supported. Please keep in mind, when you are setting assignments regex\
Links:
- https://github.com/google/re2/wiki/WhyRE2
- https://github.com/google/re2/wiki/Syntax
- https://github.com/golang/go/issues/18868

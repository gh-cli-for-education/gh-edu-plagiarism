# gh-edu-plagiarism
Plagiarism is a plug-in for the gh-edu ecosystem to detect plagiarism in programming assignments

> Plagio es la copia servil o imitación torpe de un modelo, con pretensiones de originalidad.\
> Plagiarism is the slavish copying or clumsy imitation of a model, with pretensions of originality.\
> -- <cite>Correa y Lázaro</cite>

## How does it work?
Plagiarism depends on the Stanford service MOSS, it clones all the repositories related to an assingment in an organization and sends it to MOSS service. Thereupon, it sends the result to a python script (mossum) that generates a graph.

## Installation
### Dependencies
- [MOSS](https://theory.stanford.edu/~aiken/moss/) script
    - Make sure it can be executed ``chmod ug+x moss`` 
- Perl
- Python 3
- [mossum](https://github.com/hjalti/mossum) script installed

### As a [gh-edu](https://github.com/gh-cli-for-education/gh-edu) plugin:
1. Install as a ``gh-edu`` extension ``gh edu install plagiarism``
2. Move the moss script to the root directory ``mv moss ~/.local/share/gh/extensions/gh-edu-plagiarism``

### Stand-alone:
1. Get the binary on releases or clone the repo and compile it (You will need go 1.18 or more recent)
2. Move the moss script to the same directory of the binary

## Known limitations:
Go uses google's re2 regular expression engine, which have been designed for security and predictable performance
Sadly enough this means that lookarounds are not supported. Please keep in mind, when you are setting assignments regex
Links:
- https://github.com/google/re2/wiki/WhyRE2
- https://github.com/google/re2/wiki/Syntax
- https://github.com/golang/go/issues/18868

## TODO
- [ ] Add more CLI options

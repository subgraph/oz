# gosecco - a library for parsing and managing seccomp-bpf rules

[![Build Status](https://travis-ci.org/twtiger/gosecco.svg?branch=master)](https://travis-ci.org/twtiger/gosecco)
[![Coverage Status](https://coveralls.io/repos/github/twtiger/gosecco/badge.svg?branch=master)](https://coveralls.io/github/twtiger/gosecco?branch=master)
[![GoDoc](https://godoc.org/github.com/twtiger/gosecco?status.svg)](https://godoc.org/github.com/twtiger/gosecco)

gosecco is a project to provide a full stack of tools necessary for working with SECCOMP BPF rules from Golang. The primary pieces of functionality are the parser and compiler - but the project also supports a rudimentary assembler and disassembler. It also supports an emulator that can be tweaked to provide output on whether your rules actually do what you think they should do or not. None of these tools are exposed as command line tools - they are meant to be used as libraries for higher level applications and systems.

gosecco is only compatible with Linux 3.7 and above. It has only been tested with Golang 1.6, and it assumes an amd64 architecture, although that is likely to change.

The language that gosecco parses and understands is documented in https://github.com/twtiger/gosecco/blob/master/docs/seccomp-policy-language.md.

## Libraries

gosecco is composed of several smaller libraries that provides different parts of the functionality. The gosecco package exposes the core functionality - see the godoc for more info: https://godoc.org/github.com/twtiger/gosecco.

The specific libraries used are these:

### asm

The asm package is mostly a self contained package that can be used to generate a simple form of BPF assembler, and read the same form of assembler into and out of slices of unix.SockFilter.

### checker

The checker package provides for type checking of a finished parse tree - but also makes sure that certain semantic constraints are fulfilled, such that there is not more than one rule for a specific syscall, or that all the syscalls referred actually exist.

### compiler

The compiler will take a parse tree and generate optimized BPF code in the form of a slice of unix.SockFilter - the intention is that the output of the compiler should be ready to install for a running program. The compiler doesn't implement many optimizations by itself, but it does try to be clever with jump layouts and so on. Simplification and normalization of the tree will already be done before the compiler starts working.

### constants

A helper package that contains many well known constants from the Linux environment, so that these are available to profiles written for seccomp.

### data

This package only contains the definition for the Seccomp Working memory data set, and is a helper package for the other packages.

### emulator

An emulator that takes a set of rules and an instance of working memory and executes the instructions therein. The emulation is extremely slow and obvious in order to make it easier to understand the implementation - this tool is primarily there as a basis for experiments and further evolution.

### parser

The parser is divided up into a tokenizer implemented using Ragel and a very simple recursive descent parser. The language parsed is described in the document referred to above. The output will be a raw policy document where macro definitions and rule definitions appear in the order they were defined.

### precompilation

The precompilation package contains some checks that make sure that everything is ready for being compiled. It doesn't provide error messages for users of packages, but for implementors. Basically speaking, if this ever triggers, it's because someone has wired something wrong.

### simplifier

The simplification phase takes a tree and tries to do as much optimization as possible before hand. This means basically reducing all arithmetic expressions as much as possible based on constants. We don't do more complicated optimizations such as reorderings or inversions of mathematical operations - we simple execute as much as possible beforehand. THe assumption is that there are no free variables or calls at this stage.

### tree

The tree defines the expression types and all subnodes of the AST. It also defines a Visitor that can be used to provide functionality on the AST.

### unifier

The unifier takes the set of rules and zero or more lists of macro definitions and resolves all free variables in the set of rules by replacing them with their macro content. The output will be a tree that is fit for simplification, type checking and compilation.

## Flow of execution

In general, this library will work by taking a file of definitions, parse it, compile it and install it. The specific flow of events looks like this:

- First, the file will be parsed
- Then, the unifier will take all the macro definitions and make sure the final tree is complete
- After that, the type checker will run to make sure everything looks correct
- Then, we use the simplifier to optimize and make everything smaller
- After simplification, the tree should be ready for compilation. The precompilation package checks that and ensures we are all good.
- Finally, the compiler takes the tree and turns it into bytecode
- Optionally, at this point we will install the bytecode into a running process using either the seccomp or the prctl system call.

The library can also check whether seccomp is supported. It supports the separation of macros and rules into several files. This composition cannot happen inside the files, but has to be done by the calling library. This allows for shared macros and rules. The language also supports default positive and negative actions, such that it's clear from the file itself whether it's a blacklist or a whitelist, for example. These default actions can also be specified programmatically. Finally, each rule can have custom positive or negative actions if needed.

Refer to the godoc for the API - we hope to have some usage examples up as soon as the library is finished.

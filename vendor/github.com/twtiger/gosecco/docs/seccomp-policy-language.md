# Seccomp definition policy language

## Top level syntax

Each line is its own unit of parsing - there exists no way of extending expressions over multiple lines.

Every line can be one of several types - specifically, they can be assignments, rules or comments.

In general, each line will be parsed and understood in the context of only the previous lines. That means that variables and macros have to be defined before used. This also stops recursive actions from being possible.

## Comments

A comment will start with a literal octothorpe (#) in column 0, and continues until the end of the line
No processing of comments will happen.

## Default actions

Each rule can generate a positive or a negative action, depending on whether the boolean result of that rule is positive or negative. When compiling the program it is possible to set the defaults that should be used. This might not always be the most convenient option though, so the language also supports defining default actions inside of the file itself. These can be specified by assigning the special values DEFAULT_POSITIVE and DEFAULT_NEGATIVE in the usual manner of assignment. The standard actions available have mnemonic names as well. These are  "trap", "kill", "allow", "trace". If a number is given, this will be interpreted as returning an ERRNO action for that number:

    DEFAULT_POSITIVE = trace
    DEFAULT_NEGATIVE = 42

It is suggested to define these at the top of the file to minimize confusion. It is theoretically possible to change the default actions through the file, but that is discouraged, and the result is undefined.

DEFAULT_POSITIVE and DEFAULT_NEGATIVE act on a per-line level - they only trigger if the syscall is matched. So if you have a policy file where no actions match, you might want to customize this behavior as well. That is done with a third special variable named DEFAULT_POLICY - and it acts the same way as the other two.

## Assignments
  
Assignments allow the policy writer to simplify and extract complex arithmetic operations. The operational semantics of the assignment is as if the expression had been put inline at the place where the variable is referenced. The expression defining the variable has to be well formed in isolation, but can refer to previously defined variables. The compiler will perform arithmetic simplification on all expressions in order to reduce the number of operations needed at runtime.

    var1 = 412 * 3

## Macros

If a variable expression refers to any of the special argument variables, or contains any boolean operators or the return operator, the assignment will instead refer to a macro. The operational semantics for a macro is the same as for variables. Any expression that refers to a macro becomes a macro.

Macros can take arguments - the argument list follows the usual rules and the evaluation will use simple alpha renaming before compilation.

    var2 = arg0 == 5 && arg1 == 42; return 6
    f(x) = x == 5
    g(y) = y == 6

    read: f(arg0) || g(arg0) || g(arg1) 
    read: var2

## Rules

A rule can take several different forms. Each rule will be for one specific systemcall. That systemcall will be referred to by its common name. There can only be one rule per systemcall for each policy file - except if they are equal. A rule can result in either a boolean result, or a direct return action.
If a boolean result happens, the rule will generate a return action based on the DEFAULT positive or negative action for that policy file. Specifically, a positive result from the rule, will return the DEFAULT POSITIVE action, and the negative result will return the DEFAULT NEGATIVE action.

There are several different format for rules. They all start with the name of the system call, followed by possible spaces, followed by a colon and possible spaces. The first form takes a boolean expression after the colon, and will generate a positive or negative result depending on the outcome of that boolean expression:

    read: arg0 == 1 || 42 + 5 == arg1

The second form allows you to return a specific error number when a system call is invoked:

    read: return 42

The third form combines these two, in such a way that if the given expression is positive, the default positive action will be returned, but if negative, the error number specified by return will be used:

    read: arg0==1; return 55

Rules can specify their own custom positive and negative actions that differ from the default. This uses the same naming convention as the default actions described above. The syntax for describing them is simple:

    read[+trace, -kill] : 1 == 2
    read[+42] : arg0 == 1
    read[-55] : arg0 > 1
  
The order of the actions is arbitrary, and either part can be left out. The plus sign signifies the positive action, and the minus the negative action. If no actions are specified, the square brackets can be left off, and the default actions for the file will be used.

## Syntax of numbers

Numbers can be represented in four different formats, following the standard conventions:

Octal: 0777
Decimal: 42
Hexadecimal: 0xFEFE or 0XfeFE

All numbers represent 64bit unsigned numbers. Negative numbers can only be represented implicitly, through arithmetic operations.

The BPF calculations run on a 32 bit machine, which means that using the full range of 64bits is not possible at runtime. However, all arithmetic operations that can be evaluated at compile time will end up doing the correct thing. As an example, this comparison:

    arg0 == 1 << 56

Will calculate 1 << 56 at compile time, and generate a comparison of both the upper and lower half of arg0. In general, these rules can lead to inconvenient effects - the compiler tries its best to warn in these circumstances, but it is something to be wary of.


## Arguments

System calls can take up to 6 arguments. These will be specified as arg0, arg1, arg2, arg3, arg4 and arg5. Internally, these will always be 64bit numbers. However, this causes a certain amount of awkwardness when it comes to working with them on a machine that only supports 32 bit numbers. So the rules are like this:

When an argument is directly on the left or right hand side of a comparison, the compiler will automatically extend the comparison to the upper half of the argument, such as:
`arg0 == 32`
will actually generate code to check that the lower half of arg0 is equal to 32 and the upper half is equal to 0. For comparisons, similar things happen:
`arg0 < 32`
will generate code to ensure that the upper half is 0, and the lower half is less than 32.
Comparing two arguments directly will also generate comparisons of both the upper and lower half of the arguments.

However, these methods only work if no arithmetic operations have been applied to the argument. Because of this, the language prohibits arithmetic operations on the full argument values, since they can't be encoded safely. In order to access flags or other things on the upper half of arguments, we support loading specifically the upper or lower part of the argument. This will be loaded as 32bits.. The syntax for loading the upper half is argH0, argH1, argH2, argH3, argH4 and argH5, and the lower part argL0, argL1, argL2, argL3, argL4 and argL5. 

## Syntax of expressions

The arguments to unary or binary operators can be any VALUE, where VALUE is defined to either be one of the argument names, an explicit number, or another expression.

### Arithmetic

All standard arithmetic operators from C are available and follow the same precedence rules. Specifically, these operators are:
- Parenthesis
- Plus (+)
- Minus (-)
- Multiplication (*)
- Division (/)
- Binary and (&)
- Binary or (|)
- Binary xor (^)
- Binary negation (~)
- Left shift (<<)
- Right shift (>>)
- Modulo (%)

### Boolean operations

The outcome of every rule will be defined by boolean operations. These primarily include comparisons of various kinds. Boolean operations support these operators:
- Parenthesis
- Boolean OR (||)
- Boolean AND(&&)
- Boolean negation (!)
- Comparison operators
  - Equal (==)
  - Not equal (!=)
  - Greater than (>)
  - Greater or equal to (>=)
  - Less than (<)
  - Less than or equal to (<=)
- Inclusion:
  in(arg0, 1,2,3,4)
  notIn(arg0, 1, 2, 3, 4)
  the in/not in operators are not case sensitive. Any valid value or name can be used inside the brackets. Values have to be separated
  by commas, and arbitrary amount of whitespace (tabs or spaces). The in/notIn operator is the function like application that is not actually a function

These can all be arbitrarily nested. The precedence between boolean operators and arithmetic operators differ from those in most languages. Specifically, the precedence prefers all boolean operations before all arithmetic operations. In real terms, that means the precedence schedule looks about like this:

01. Boolean OR: ||
02. Boolean AND: &&
03. Equality expressions: ==, !=
04. Relativity expressions: <, <=, >, >=
05. Binary or: |
06. Binary xor: ^
07. Binary and: &
08. Shift expressions: <<, >>
09. Additive expression: +, -
10. Multiplicative expression: *, /, %
11. Unary expression: !, ~
12. Primary expression: argument, variable, call, parenthesised expression, in, notIn

As a special case, the string "1" can be used as a short form of specifying the allow case for a rule. No other symmetric values are valid in the same setting.

    read: 1

### Compatibility note

The current language as defined is almost completely backwards compatible with the previous seccomp definition language, with one big difference. The bitset comparison operator & has been moved to be &? instead. This was done to remove ambigous parsing rules.

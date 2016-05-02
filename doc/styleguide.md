# Chain Style Guide

- [Introduction](#introduction)
- [Code Review Principles](#code-review-principles)
- [Code Review Mechanics](#code-review-mechanics)
  - [Branches](#branches)
  - [Iteration](#iteration)
  - [Commit Messages](#commit-messages)
  - [Trello](#trello)
  - [Testing](#testing)
  - [Documentation](#documentation)
  - [A Tough Example](#a-tough-example)
  - [Why?](#why)
- [Code Style Mechanics](#code-style-mechanics)
  - [Go](#go)
  - [Java](#java)
  - [Ruby](#ruby)

## Introduction

The ultimate goal is to ship a high-quality product while leading
high-quality lives. The question is how.

Concretely, for us, shipping a high-quality product primarily means
building software systems. Our worst enemy in this game is complexity.
Our strongest ally is clarity: simplicity, readability, organization,
documentation. A tangled morass, even if perfectly correct to start,
will become a cesspool of bugs in short order. But we can fix any
broken behavior or performance problem if only we understand the code.

<img src="churchillmemo.jpeg" alt="Churchill memo" width="50%" align="right">

Churchill had something to say about readability and organization.

His memo applies to software almost word-for-word. Note even the point
about “refactoring” into appendices to handle essential complexity.

The issue of “padding” phrases recalls our stylistic conventions for
naming symbols. Overly-long names, like wooly sentences, carry little
or no additional information. The Unix library function is not
waitforpid, but waitpid, which, in context,<sup id=a1>[1](#f1)</sup>
works better. It carries just as much information, and it is more
legible.

The list of virtues above includes simplicity, readability,
organization, and documentation, but not flexibility. It is
deliberately excluded. What’s most important is not how easy the
system is to change, but how easy it is to _figure out_ what to
change. An ostensibly inflexible but simple system wins over one with
a million “flexible” layers of indirection. For example, despite bzr’s
[vaunted clean and modular
design](https://www.jelmer.uk/pages/bzr-a-retrospective.html#hard-to-land-patches):

> Most Git users know the basics of its file format or can learn it in
> a day; it is much easier for them to find a bug in one of gits few
> source files than it is to understand the various layers involved in
> Bazaar.

Much more can be said about the philosophy of the code itself, but how
does this apply to the process of writing it? How should we organize
our patches as the code evolves?

## Code Review Principles

What we want during code review:

- **Good design** Keep the code clear, don’t let the results of haste,
laziness, or fuzzy thinking creep into the codebase.
- **To catch bugs** Of course. Mo’ eyes mo’ betta.
- **To transfer knowledge** Deep understanding of any part of our
codebase should not be confined to a single engineer.
- **To discuss design** Within limits, a code review is an acceptable
place to discuss the design of a new feature, with the code under
review serving as a sketch of one possible solution.
- **To maintain house style** Make sure we all hew to our style
guidelines for maximum maintainability.
- **Thorough and clear documentation** When a reviewer has trouble
following the logic of a change, add documentation.
- **Debuggability across both space (packages) and time (git bisect)**
Don’t let snowballing changes conceal the origin of a bug.
- **Expediency** We have to ship!

These are all meant to help accomplish the ultimate goal up there.

We have plenty of ideas for processes meant to further these goals.
Each is a tradeoff. It helps in some ways, hurts in
others.<sup id=a2>[2](#f2)</sup>

Sometimes these goals are in tension—keeping our quality standards too
high would mean we never ship anything—but often they are in harmony.
A simpler system is easier to understand, and therefore easier to
change. This is a boon not only to quality, but to shipping reliably
after weeks and months and years.

So what makes a good patch or a good review? How can we satisfy the
above concrete goals? It depends on who you are.

What makes a good patch, as a reviewer:
- Atomic units of work. Is this patch small? Is this patch the
smallest cohesive change that can be made?
- Flow. Is there a line of reasoning I can follow? Do I know where to
jump into a change?
- Context. Do I know why this patch has been written? Am I familiar
with the code being changed?

What makes a good patch, as a patch author:
- Non-conflicting. Does this patch conflict with work that another
engineer is doing?
- Non-blocking. Is this PR blocking any of the other work I’m doing?
This can conflict with the desire for atomic units of work. When a
project is broken down into its smallest units of work, these units
often build on one another. However, we have chosen to prioritize
atomicity because it lets the team as a whole move more quickly.

What makes a good review, as a patch author:
- Speed. How quickly can the reviewer provide feedback?
- Progress. Will I have to rewrite something more than once? Will
another reviewer jump in and contradict the first?

An “atomic” patch is both complete and minimal.

The patch should ideally stand on its own and not leave the codebase
inconsistent. Calling a new function without adding it won’t even
compile, and will be rejected by the testing bot. But adding a new
function without calling it anywhere also creates inconsistency: the
new function is dead code. On occasion, this is appropriate for the
sake of expediency, but in this case be sure to leave a TODO comment
explaining the situation.

But what about adding two new functions and calling both of them at
once? This might be necessary, but unless the functions are mutually
recursive, it’s likely that one or the other can be added first,
resulting in two smaller patches.

In that case, deciding whether they should be added separately
requires a judgement call. The primary criterion should be
comprehensibility. Would they be easier to understand together, or
apart? (This can be in tension with expediency, this is dicussed more
in the “tough example” below.)

## Code Review Mechanics

Here are the guidelines for how to write and review and land a patch.
They’re meant to serve the principles discussed above. They aren’t
absolute rules, but if you don’t follow them, expect to be questioned
by your colleagues on why.

### Branches

When you work on a task, open a topic branch. When you’re ready to
merge it, push it to GitHub and open a pull request. It’s ok to rebase
liberally, and even squash commits on your topic branch, but it’s
usually clearer to just add new commits on top, especially if you’re
collaborating on the branch.

If you need to incorporate changes made on main after you started your
branch, prefer rebasing your branch on top of the new main rather than
merging main into your branch. Also, enable the [rerere
feature](https://git-scm.com/blog/2010/03/08/rerere.html)

```
[rerere]
    enabled = true
```

in your ~/.gitconfig.

### Commit Messages

Flow and context are aided by good documentation. Sometimes they are
obvious from the code, but even well-written code often needs more.

Commit messages should have the following format:

- Subject line. The first line is a “subject” or “short description”.
It starts with the import path of the package (or the directory or
whatever) that the patch is primarily concerned with (leaving off the
leading “chain/”) and a colon. If there is an interface change, the
patch will likely touch other packages to conform to the new
interface, but the name listed should be where the action is. The
filesystem hierarchy is irrelevant to this; for example, if you’re
changing a function in package api/txdb, and it ripples out to
api/asset and api itself, the name to list is api/txdb. The rest is an
ultra-brief statement of the purpose of the patch. Shoot for <50
characters total, but this can be hard with long import paths so don’t
worry too much.

- Blank line following the subject.

- Details. This is a long description of what the change is, why we
need it, any necessary background information, relevant future plans,
a rationale for any technical tradeoffs or other choices made, what
the alternatives were, or any other notes you think might be good to
include. This can be empty if everything is obvious<sup
id=a3>[3](#f3)</sup> just from looking at the diff, or it can be very
long. Hard wrap paragraphs to something reasonable (between 65-75
columns). The rule of thumb is to describe it as you would to a
coworker sitting next to you (so you should assume they know all the
general CS background knowledge and have general familiarity with our
codebase as a whole) who doesn’t know any of the specific context or
motivation of this patch (so don’t assume intimate familiarity with
this region of the codebase, especially if it’s intricate). Sometimes
a patch requires studying the existing code or careful planning before
you even start writing new code. Don’t make the reviewer redo your
research just so they can understand your patch. Spell it out. Be
liberal including links to other resources.

If the change is part of a group of related patches, by all means do
say so, but don’t lead with that. The patch description’s first job is
to say what the patch does. Again, consider someone revisiting this
patch after six months. It will be helpful to know that it’s part of a
set, but that’s probably not the first question they’ll ask.

More generally good advice on commit messages:
- https://robots.thoughtbot.com/5-useful-tips-for-a-better-commit-message
- http://mislav.net/2014/02/hidden-documentation/

Finally, list the reviewers and the pull requests or issues (can be
several) where this patch was discussed. Use GitHub’s “closes #nnn”
notation to tell GitHub to close the ones that need to be closed. This
part generally needs to be added at the end of the review process, and
it’s mostly automated by
[git-land](https://github.com/chain-engineering/chain/blob/main/bin/git-land)
(see below).

### Trello

If the change is for a card in Trello, put a link to it (either the
pull request or the commit) in the card. In general, liberally link
(in both directions) related things that have URLs. Where possible,
the tools will help remind you or do it for you.

### Testing

*Write tests*. Especially regression tests when fixing a bug.

(Pure refactoring often doesn’t need new tests. Deleting code rarely
requires adding tests. But almost every other code change ought to be
accompanied by either new tests or modifications to existing tests.)

### Documentation

When designing a major new subsystem or a particularly intricate
algorithm, write up your design in a Readme file or large block
comment in one of the main source files.

Most of our interfaces, even internal ones, are documented. When
adding new interface surface area, document it. When modifying
existing interfaces, update the existing documentation. This is easy
to forget. For example, a package overview might say “the functions in
this package operate on strings”, and you might add a new function
that operates on byte slices. You’ve dutifully documented your new
function, but the package overview is now wrong. Reviewers: assist
your author in this. Be on the lookout for stale documentation.

### Checklist

Before you submit your patch for review, look at the diff. Consider
running through a quick mental checklist of all the things discussed
above before submitting your patch for review:

- Did I write tests and update docs?
- Can it be any clearer?
- Can I delete more lines of code?
- Does the diff overall look reasonable?
- Is there any cruft like stray blank lines or debugging output or
irrelevant trivial refactorings?
- Should the change be broken into multiple meaningful pieces?
- Is this change really necessary at all?

(You might be surprised how often the answer to that last one is
“no”. 😉)

### Iteration

When you’re ready for review, put the magic word “PTAL” in a comment
on the pull request. This stands for “please take a(nother) look” and
tells the world you are ready for some (more) review. You’re
encouraged also to ask a specific person or two if you know they have
relevant expertise in the subject matter or region of code you’re
touching. Your reviewers will comment “👀” to indicate they’re
looking at it, they’ll add their comments and questions, and you’ll
update your patch. When the reviewers are satisfied, they’ll write
“LGTM”, for “looks good to me”.

Then it’s time to land the change. Our tool
[git-land](https://github.com/chain-engineering/chain/blob/main/bin/git-land)
automates much of the work of landing. First it checks a couple of
preconditions, such as whether at least one person has LGTM’d the pull
request, and whether the automated testing bot has marked it as
passing. A reviewer can veto the patch by writing “NOT LGTM” if they
think landing it as-is would be a problem. Assuming all its checks
pass, git-land squashes the branch down to a single commit, pushes it
to branch main, and cleans up the topic branch. (Using git-land
ensures each commit on main has a single parent. Merge commits are
prohibited. They contribute no useful information and add clutter.)

### A Tough Example

Consider a large patch comprising several related but distinct new
features. It would be preferable to introduce those features one by
one in a series of related patches. However, it might not be obvious
how to pull them apart because of a complex web of interdependencies.
Doing so might even require some refactoring of the newly-introduced
code.

In that case, teasing them apart becomes doubly important. The same
web of dependencies will also make the patch more difficult to review,
and the resulting codebase more difficult to debug. (Remember, even if
this code contains no bugs, we’ll still have to trace through it to
solve other problems.)

In practice, it seems to be rare for a patch that ought to be split up
to be so intricate that that would be infeasible. Even if the
dependencies look like a mess, you can always find one or two leaves.
Don’t worry about the rest of the patch, just pull the obvious leaves
out into their own patches, and land them. Then look at what’s left.
It’ll be slightly easier to understand. It will have new leaves. Those
new leaves might have been impossible to identify before, but now they
should be clear. Repeat this process until what’s left is simple and
minimal—appropriate for a single patch.

Try to do it as you go. If you’re working on one change, and it
uncovers something that needs to happen first, set aside the current
work and do that other thing. Classic yak shaving. But sometimes it’s
not clear what exactly needs to be done until it’s all complete. Even
then, after writing the full set of changes all in one messy topic
branch, consider going through that iterative process above.

The benefit here isn’t only for posterity. Doing it right makes your
own changes easier to write and to review and to land.

### Why?

Producing a series of patches touching closely-related code results in
intermediate states of the codebase. Why bother to polish these
intermediate states? Why, when we know we’ll be changing the code
again, mere minutes later, in the next patch?

Recall the goals for the patch process: good review, good docs,
debuggability, and expediency.

Expediency is the puzzle. It’s all too easy to find ourselves in that
tough situation described above. On the surface, it might seem most
expedient to take such a patch, once written, and review it as-is,
without attempting to split it up. But this comes at a high cost. It’s
likely that some parts of the patch will need deeper discussion that
others, and it’s helpful to have discussion in a focused context
without distraction. Attempting deep discussions directly on the
larger patch can take longer overall, and will probably be less
effective. And if the patch get broken up, and the deep discussion
takes place on patches later in the series, then so much the better
for expediency: the earlier patches will land sooner, without waiting
for that long discussion.

As for the rest, having clean intermediate states helps document how
we got here, helps us to debug, and helps us with review. It’s not
only about the current code. The history of how we got here provides
valuable documentation—it’s just as important.

## Code Style Mechanics

### Go

We follow the advice and conventions in these documents from the Go
team:
- https://golang.org/doc/effective_go.html
- https://github.com/golang/go/wiki/CodeReviewComments

Where possible, we enforce style rules mechanically.

You’ll need some basic tools: gofmt, goimports, go vet, and golint.
Some of them come with the Go distribution, but a few must be
installed separately. Your text editor might come with features to
make editing Go more convenient.

```
$ go get\
	golang.org/x/tools/cmd/goimports\
	github.com/golang/lint/golint\
```

Reserve the `if x := f(); ... {` form for when f is called primarily
for its return value, and has no significant side effects. Otherwise,
put `x := f()` on its own line.

External dependency code is copied (“vendored”) into our repository so
that it’s guaranteed to remain available and so that it changes only
by our deliberate action. The details of how we do this might change,
but the principle of vendoring is part of our style guide.

### Java

[TBD]

### Ruby

`.tap` blocks should be only one line

[should we consider using some subset of https://github.com/styleguide/ruby?]

----

<b id=f1>1</b>: But do note the importance of context! Leaving out
connecting words like “for” works in the context of strong
conventions, plus the assumption that we’ll use the convention often
enough to outweigh the cost of learning it.
[↩️](#a1)

<b id=f2>2</b>: And the tradeoff differs depending on context: some
processes work well only among certain others. It’s generally not
possible to evaluate a potential process change in isolation. We must
consider what other changes might need to go along with it so the
whole system is closer to the goal. [↩️](#a2)

<b id=f3>3</b>: Obviousness correlates with brevity, but of course
there are plenty of exceptions. Consider a subtle one-line change in
the heart of a distributed consensus algorithm, or a system-wide
renaming of one widely-used symbol. [↩️](#a3)

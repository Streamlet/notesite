# Overview

### How to maintain notes in file system

Just put all notes in an ordinary directory ("note_root").
Sub-directories is supported for categorizing notes.
The directory tree could be like this:
```
+ note_root
    + note1.md
    + note2.md
    + category1.md
        + note3.md
        + note4.md
    + category2.md
        + note5.md
        + note6.md
        + sub-category
            + note7.md
```

See [note](note) section for details.

### How to show notes with web pages

We support HTML templates. There 3 kinds of templates:
* index
* category
* content

For default config, we need a directory called "template_root", in the following structure:
```
+ template_root
    + index.template.html
    + category.template.html
    + content.template.html
    + 404.html (optional)
    + 500.html (optional)
    + assert (optional)
        + files in asserts
```

"index.template.html" is used for "note_root" directory,
"category.template.html" is used for all sub-directories in "note_root",
and "content.template.html" is used for note files.

See [template](template) section for details.

### Is there a DEMO site?
Yes. This site itself is served by notesite.
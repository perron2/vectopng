vectopng
========

A simple tool to convert Android vector drawables to PNG image files.
Only `path` elements are currently supported, `groups` and `clip-path`
elements cannot be used.

```
Usage: vectopng [options] <vector-image-input> [<png-image-output>]

  -color value
    	Defines an (A)RGB value for a color name (name=#(a)rgb|(aa)rrggbb)
  -colors string
    	Defines an Android color resource file to be parsed for color definitions
  -height float
    	Overrides the canvas height attribute of the vector drawable
  -ios
    	Generates three resolutions of the image (adds @2x and @3x versions)
  -scale float
    	Scales the image by the given factor (default 1)
  -version
    	Shows the program version
  -width float
    	Overrides the canvas width attribute of the vector drawable
  -x float
    	Translates the image in x direction
  -y float
    	Translates the image in y direction
```

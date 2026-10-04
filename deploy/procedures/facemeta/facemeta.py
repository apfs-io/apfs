#!/usr/bin/env python3

# @link https://realpython.com/face-recognition-with-python/
# haarcascade_frontalface_default.xml
# haarcascade_eye.xml
#
# $ brew tap brewsci/science
# $ brew install cmake pkg-config
# $ brew install jpeg libpng libtiff openexr
# $ brew install eigen tbb
# $ brew install opencv3 --with-contrib --with-python3 --HEAD
#
# $ cat requirements.txt
#   opencv-python~=4.2.0.32
#   opencv-contrib-python~=4.2.0.32
#

import sys
import cv2
import os
import json

xmlpath = os.path.join(os.path.dirname(cv2.__file__), 'data')

# Get user supplied values
imagePath = sys.argv[1]
cascPath = sys.argv[2] if len(sys.argv)>2 and sys.argv[2] else 'haarcascade_frontalface_default.xml'

# Create the haar cascade
faceCascade = cv2.CascadeClassifier(os.path.join(xmlpath, cascPath))

# Read the image
image = cv2.imread(imagePath)
gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
height, width, channels = image.shape

# Detect faces in the image
faces = faceCascade.detectMultiScale(
    gray,
    scaleFactor=1.2,
    minNeighbors=5,
    minSize=(30, 30),
    flags = cv2.CASCADE_SCALE_IMAGE
)

boxes = []

# Draw a rectangle around the faces
for (x, y, w, h) in faces:
    boxes.append({"x": int(x), "y": int(y), "w": int(w), "h": int(h)})

print(json.dumps({
    "h":        height,
    "w":        width,
    "channels": channels,
    "boxes":    boxes,
}))

# # Draw a rectangle around the faces
# for (x, y, w, h) in faces:
#     cv2.rectangle(image, (x, y), (x+w, y+h), (0, 255, 0), 2)

# cv2.imwrite(os.path.join(os.path.dirname(imagePath), 'out.jpg'), image)

# # cv2.imshow("Faces found", image)
# # cv2.waitKey(0)

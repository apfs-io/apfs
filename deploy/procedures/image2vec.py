#!/usr/bin/env python3
# @link https://github.com/christiansafka/img2vec
#
# pip3 install torch img2vec_pytorch Pillow

import torch
import json
import sys
from img2vec_pytorch import Img2Vec
from PIL import Image
# from sklearn.metrics.pairwise import cosine_similarity

if not sys.warnoptions:
    import warnings
    warnings.simplefilter("ignore")

# Initialize Img2Vec with GPU
img2vec = Img2Vec(cuda=torch.cuda.is_available(), model='resnet-18')

imagePath = sys.argv[1]
# Read in an image
img = Image.open(imagePath)
# Get a vector from img2vec, returned as a torch FloatTensor
vec = img2vec.get_vec(img, tensor=True)

l = vec.view(-1).numpy().tolist()
print(json.dumps(l))

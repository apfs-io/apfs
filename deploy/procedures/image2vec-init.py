#!/usr/bin/env python3
#
# Execute all init file before run
# $ find /procedures/*-init* | xargs -I '{}' bash -c '{}'

from img2vec_pytorch import Img2Vec
import torch

Img2Vec(cuda=torch.cuda.is_available(), model='resnet-18')
print("image2vec inited")

#!/bin/bash

cd ..
find . -name "*.sh" -exec dos2unix {} \;

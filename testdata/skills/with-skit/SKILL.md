---
name: with-skit
description: Skill with skit metadata.
metadata:
  skit:
    version: 1.2.0
    dependencies:
      - source: github:example/pdf-core
        ref: v1.2.0
        skill: pdf-core
        optional: true
    requires:
      bins:
        - qpdf
      anyBins:
        - pdftotext
        - mutool
      env:
        - PDF_API_KEY
    platforms:
      os:
        - linux
        - darwin
    keywords:
      - pdf
---
# With skit

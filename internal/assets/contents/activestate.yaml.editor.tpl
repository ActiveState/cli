scripts:
  - name: helloWorld
    if: ne .OS.Name "Windows"
    value: echo "Hello World!"
  - name: helloWorld
    if: eq .OS.Name "Windows"
    value: echo Hello World!
  - name: intro
    if: ne .OS.Name "Windows"
    value: |
      echo "Your runtime environment is now ready!"
      echo ""
      echo "To see how scripts work and add your own, open up the activestate.yaml file with your editor."
  - name: intro
    if: eq .OS.Name "Windows"
    value: |
      echo Your runtime environment is now ready!
      echo.
      echo To see how scripts work and add your own, open up the activestate.yaml file with your editor.
events:
  - name: ACTIVATE
    value: $scripts.intro

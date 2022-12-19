scripts:
  - name: helloWorld
    constraints:
      os: macos,linux
    value: echo "Hello World!"
  - name: helloWorld
    constraints:
      os: windows
    value: echo Hello World!
  - name: intro
    constraints:
      os: macos,linux
    value: |
      echo "Your runtime environment is now ready!"
      echo ""
      echo "To see how scripts work and add your own, open up the activestate.yaml file with your editor."
  - name: intro
    constraints:
      os: windows
    value: |
      echo Your runtime environment is now ready!
      echo.
      echo To see how scripts work and add your own, open up the activestate.yaml file with your editor.
events:
  - name: ACTIVATE
    value: $scripts.intro
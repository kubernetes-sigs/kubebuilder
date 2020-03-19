# Enabling shell autocompletion
The Kubebuilder completion script for Bash can be generated with the command `kubebuilder completion bash` as the Kubebuilder completion script for Zsh can be generated with the command `kubebuilder completion zsh`. 
Note that sourcing the completion script in your shell enables Kubebuilder autocompletion. 

<aside class="note">
<h1>Prerequisites for Bash</h1>

The completion Bash script depends on [bash-completion](https://github.com/scop/bash-completion), which means that you have to install this software first (you can test if you have bash-completion already installed). Also, ensure that your Bash version is 4.1+. 

</aside>


- Once installed, go ahead and add the path `/usr/local/bin/bash` in the  `/etc/shells`.

    `echo “/usr/local/bin/bash” > /etc/shells`

- Make sure to use installed shell by current user.

    `chsh -s /usr/local/bin/bash`

- Add following content in /.bash_profile or ~/.bashrc

```
# kubebuilder autocompletion
if [ -f /usr/local/share/bash-completion/bash_completion ]; then
. /usr/local/share/bash-completion/bash_completion
fi
. <(kubebuilder completion)
```
- Restart terminal for the changes to be reflected.

<aside class="note">
<h1>Zsh</h1>
Follow a similar protocol for `zsh` completion.
</aside>

# 开启 shell 自动补全

Kubebuilder 的 Bash 补全脚本可以通过命令 `kubebuilder completion bash` 来自动生成，Kubebuilder 的 Zsh 补全脚本可以通过命令 `kubebuilder completion zsh` 来自动生成。需要注意的是在你的 shell 环境中用 source 运行一下补全脚本就会开启 Kubebuilder 自动补全。

<aside class="note">
<h1>Bash 前提条件</h1>

Bash 脚本补全依赖于[bash-completion](https://github.com/scop/bash-completion)，这也就意味着你不得不首先安装此软件（如果你已经安装了 bash 补全，你可以进行测试）。此外，确保你的 Bash 版本是 4.1+。

</aside>


- 一旦安装完成，要在 `/etc/shells` 中添加路径 `/usr/local/bin/bash`。

    `echo “/usr/local/bin/bash” > /etc/shells`

- 确保使用当前用户安装的 shell。

    `chsh -s /usr/local/bin/bash`

- 在 /.bash_profile 或 ~/.bashrc 中添加以下内容：

```
# kubebuilder autocompletion
if [ -f /usr/local/share/bash-completion/bash_completion ]; then
. /usr/local/share/bash-completion/bash_completion
fi
. <(kubebuilder completion)
```
- 重启终端以便让修改生效。

<aside class="note">
<h1>Zsh</h1>
`zsh` 补全可以参考上述流程。
</aside>

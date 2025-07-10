export GPG_TTY="$( tty )"

#COMPOSER
export PATH=$PATH:~/.composer/vendor/bin

#MISCELLANEOUS
alias weather="curl http://wttr.in"
alias code="cd ~/Sites"
alias api="cd ~/Sites/praesenx/api"
alias web="cd ~/Sites/praesenx/web"
alias ppath="echo $PATH | tr ':' '\n'"
alias zrestart="exec zsh --login"

#LARAVEL
alias aa="php artisan"
alias tinker='php artisan tinker'
alias l-perm="sudo chgrp -R www-data storage/ bootstrap/cache && sudo chmod -R ug+rwx storage bootstrap/cache"
alias l-clear-logs="rm -rf storage/logs/*.*"
alias l-clear-cache="php artisan cache:clear && php artisan config:clear && php artisan clear-compiled && php artisan route:clear && php artisan view:clear"
alias l-sail='[ -f sail ] && sh sail || sh vendor/bin/sail'
alias l-amf="php artisan migrate:fresh"

#PHP
alias cda="composer dumpautoload -o"
alias phpini="php -i | grep php.ini"
alias uf="./vendor/bin/phpunit --filter="
alias u="./vendor/bin/phpunit"

#GIT
alias gs="git status"
alias gaa="git add ."
alias gcc='git commit -S --amend --no-edit'
alias gc="git commit -S -a -m"
alias gcm="git checkout main && git pull"
alias gcd="git checkout develop && git pull"
alias gl="git log --graph --pretty=format:'%Cred%h%Creset -%C(yellow)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset' --abbrev-commit"
alias nah="git reset --hard && git clean -df"
alias wip="git add . && git commit -m 'wip'"
alias gclean="git fetch -p"
alias gempty="git commit --allow-empty -m 'Empty - Commit'"
alias ginit="git config --global pull.rebase true" #reconcile git diverged

# --- GO: add these lines if GO was installed outside Homebrew.
# export GOPATH=$HOME/go
# export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
# export GOROOT="$(brew --prefix go)/libexec"
# export PATH="$GOROOT/bin:$PATH"

export PATH=$PATH:$(go env GOPATH)/bin







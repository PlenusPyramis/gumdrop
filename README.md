# gumdrop

Manage DigitalOcean droplets from the command line with cloud-init.

This README is a design document, not everything works yet!!

## Requirements

 * You will need a [DigitalOcean](http://digitalocean.com/) account. 
 * The gumdrop executable.

## Configuration

gumdrop configuration is fully interactive, and you just need to answer
questions it asks of you. Each droplet will have a seperate configuration, but
all configurations are stored in one config file (`$HOME/.gumdrop.yaml`).
gumdrop will create this file for you if it does not exist.

Start by creating a new droplet configuration. Run:

```
gumdrop config create
```

 * The very first time you run this, it will ask you to enter your Personal
   Access Token. This will allow gumdrop to utilize your DigitalOcean account on
   your behalf. To get your Personal Access Token, go to your account dashboard,
   go to the API tab, and click `Generate New Token.` (See
   https://cloud.digitalocean.com/account/api/tokens) - Copy the token to your
   clipboard and paste it where gumdrop asks the question for it.

 * Note that your Personal Access Token is stored locally in
   `$HOME/.gumdrop.yaml`. Appropriate file permissions have been set so that
   only your user account (and root) can read it. You must keep this file safe,
   as it contains the key to access your DigitalOcean account.

 * Follow the rest of the questons that gumdrop asks you. It will take you
   through the following steps to create a new droplet configuration:
   
   * Droplet name
   * Droplet size (memory/cpu/disk)
   * Volumes (size in GB)
   * Floating IP address
 
 * Once you complete the questions, all configuration for the droplet is stored in `$HOME/.gumdrop.yaml`

Note that you are not paying for this droplet yet (since it is not created on
DigitalOcean yet). However, if you chose to use a Floating IP adddress, or
external Volumes, these resources ARE created, and you are now incurring charges
on your account, even when the droplet is not.

## Manage Droplets

You can list all of your droplet configurations by running:

```
gumdrop config list
```
   
Which will show you something like this:

```
+------------|-------------|--------------|--------|-----------------|--------+
|    NAME    |    SIZE     |    IMAGE     | REGION |   FLOATING IP   | STATUS |
+------------|-------------|--------------|--------|-----------------|--------+
| docker-dev | s-1vcpu-1gb | docker-18-04 | nyc3   | 174.138.118.207 | unused |
+------------|-------------|--------------|--------|-----------------|--------+
```

The `STATUS` field shows you that this droplet is unused. Unused means that only
a local configuration exists, and that no droplet has yet been created on
DigitalOcean.

To start the droplet named `docker-dev`, run:

```
gumdrop start docker-dev
```

To stop it (keeps it on DigitalOcean in stopped state, which does not save any money), run:

```
gumdrop stop docker-dev
```

To remove the droplet from DigitalOcean (but retain the local configuration), run:

```
gumdrop rm docker-dev
```

To remove the local configuration entirely, you can run:

```
gumdrop config rm docker-dev
```

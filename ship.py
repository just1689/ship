#!/usr/bin/env python3

import json
import os
import yaml
import click
import sys
from jinja2 import Environment, FileSystemLoader
from subprocess import check_call, check_output
from jsonschema import validate
env = Environment(
    loader=FileSystemLoader('.')
)

# Default binary paths (expects them to be in $PATH)
kops_bin = "kops"
terraform_bin = "terraform"

def get_config(config_path):
    with open(config_path, 'r') as kops_values_file:
        return yaml.load(kops_values_file)

def generate_terraform_config(template_path, output_path, cluster_config):
    terraform_template = env.get_template(template_path)
    terraform_config = terraform_template.render(cluster_config)
    terraform_config_path = '{}/networking.tf'.format(output_path)

    with open(terraform_config_path, 'w') as terraform_file:
        terraform_file.write(terraform_config)
        terraform_file.flush()

    return terraform_config_path

def generate_kops_config(cluster_config, kops_template_path, terraform_state_path, output_dir):
    with open(terraform_state_path, 'r') as terraform_state_file:
        tf_state = json.load(terraform_state_file)
        # There only is one module
        tf_resources = tf_state['modules'][0]['resources']

        region = '{}{}'.format(cluster_config['AWSRegion'], cluster_config['AWSAZ1'])
        cluster_config['VPCID'] = tf_resources['aws_vpc.'+cluster_config['ShortName']]['primary']['id']
        cluster_config['PublicNATGatewayID'] = tf_resources['aws_nat_gateway.public-'+region]['primary']['id']
        cluster_config['NodeSubnetID'] = tf_resources['aws_subnet.nodes-'+region]['primary']['id']
        cluster_config['PublicSubnetID'] = tf_resources['aws_subnet.public-'+region]['primary']['id']

    kops_template = env.get_template(kops_template_path)
    kops_config_path = '{}/kops.config'.format(output_dir)

    with open(kops_config_path, 'w') as kops_config_file:
        kops_config_file.write(kops_template.render(cluster_config))

    return kops_config_path

def terraform_apply(terraform_config_dir, terraform_state_path):
    check_call([terraform_bin, "apply", "-state=" + terraform_state_path, terraform_config_dir])

def terraform_destroy(terraform_config_dir):
    # Force parameter is used to avoid prompting the user for confirmation
    check_call([terraform_bin, "destroy", "-force", terraform_config_dir])

def kops_add_cluster(kops_config_path, state_path):
    check_call([kops_bin, "create", "-f", kops_config_path, "--state", state_path])

def kops_add_ssh_key(cluster_name, ssh_public_key_path, state_path):
    check_call([kops_bin, "create", "secret", "--name", cluster_name, "sshpublickey", "admin", "-i", ssh_public_key_path, "--state", state_path])

def kops_update_cluster(cluster_name, state_path):
    check_call([kops_bin, "update", "cluster", "--name", cluster_name, "--state", state_path, "--yes"])

def kops_replace_cluster(kops_config_path, state_path):
    check_call([kops_bin, "replace", "cluster", "-f", kops_config_path, "--state", state_path])

def kops_rolling_update(cluster_name, state_path):
    check_call([kops_bin, "rolling-update", "cluster", "--name", cluster_name, "--state", state_path])

def kops_destroy_cluster(cluster_name, state_path):
    check_call([kops_bin, "delete", "cluster", cluster_name, "--state", state_path, "--yes"])

def validate_file_exists(path, context):
    if not os.path.isfile(path):
        raise IOError("{} not found at {}".format(context, path))

def validate_config(config):
    try:
        with open('values.schema', 'r') as schema_file:
            values_schema = yaml.load(schema_file)
    except Exception as e:
        raise IOError("Failed to load config values schema file: " + str(e)) from e 

    validate(config, values_schema)

    validate_file_exists(config['paths']['sshPublicKeyPath'], 'SSH Public Key')
    validate_file_exists(config['paths']['terraformTemplatePath'], 'Terraform Template')
    validate_file_exists(config['paths']['kopsTemplatePath'], 'Kops Template')
    if not os.path.isdir(config['paths']['outputDir']):
        raise IOError("output directory '{}' does not exist".format(config['paths']['outputDir']))
    if 'terraform' in config['paths']:
        validate_file_exists(config['paths']['terraform'], "terraform binary")
    if 'kops' in config['paths']:
        validate_file_exists(config['paths']['kops'], "kops binary")

def set_binary_paths(config):
    if 'terraform' in config['paths']:
        global terraform_bin
        terraform_bin = config['paths']['terraform']
    try:
        check_output([terraform_bin, "--version"])
    except Exception as e:
        raise IOError("terraform executable not valid: " + str(e)) from e

    if 'kops' in config['paths']:
        global kops_bin
        kops_bin = config['paths']['kops']
    try:
        check_output([kops_bin, "version"])
    except Exception as e:
        raise IOError("kops executable not valid: " + str(e)) from e

@click.group()
@click.option('--values', default='values.yaml', help='Path to configuration values file')
@click.pass_context
def cli(ctx, values):
    ctx.obj = dict()
    try:
        config = get_config(values)
    except IOError as e:
        print("Failed to read values file:", str(e), file=sys.stderr)
        sys.exit(1)
    except yaml.YAMLError as e:
        print("Failed to parse values file:", str(e), file=sys.stderr)
        sys.exit(1)

    try:
        validate_config(config)
    except Exception as e:
        print("Config validation failed:", str(e), file=sys.stderr)
        sys.exit(1)

    set_binary_paths(config)
    ctx.obj['config'] = config

@cli.command()
@click.pass_context
def create(ctx):
    config = ctx.obj['config']
    cluster_config = config['clusterConfig']
    paths = config['paths']
    output_dir = paths['outputDir']

    try:
        generate_terraform_config(paths['terraformTemplatePath'], output_dir, cluster_config)
        terraform_state_path = '{}/terraform.tfstate'.format(output_dir)
        terraform_apply(output_dir, terraform_state_path)

        kops_config_path = generate_kops_config(cluster_config, paths['kopsTemplatePath'], terraform_state_path, output_dir)
        kops_add_cluster(kops_config_path, cluster_config['ConfigBaseURL'])
        kops_add_ssh_key(cluster_config['FullyQualifiedName'], paths['sshPublicKeyPath'], cluster_config['ConfigBaseURL'])
        kops_update_cluster(cluster_config['FullyQualifiedName'], cluster_config['ConfigBaseURL'])
    except Exception as e:
        print("Create cluster failed:", str(e), file=sys.stderr)
        sys.exit(1)

@cli.command()
@click.option('--yes', help='Answers "yes" to confirmation prompts"', is_flag=True)
@click.pass_context
def destroy(ctx, yes):
    config = ctx.obj['config']

    if not yes:
        click.confirm('Are you sure you want to destroy {}?'.format(config['clusterConfig']['FullyQualifiedName']), abort=True)

    try:
        kops_destroy_cluster(config['clusterConfig']['FullyQualifiedName'], config['clusterConfig']['ConfigBaseURL'])
        terraform_destroy(config['paths']['outputDir'])
    except Exception as e:
        print("Destroy cluster failed:", str(e), file=sys.stderr)
        sys.exit(1)

@cli.command()
@click.option('--yes', help='Answers "yes" to confirmation prompts". Use with caution.', is_flag=True)
@click.pass_context
def update(ctx, yes):
    config = ctx.obj['config']
    cluster_config = config['clusterConfig']
    cluster_name = cluster_config['FullyQualifiedName']
    paths = config['paths']
    output_dir = paths['outputDir']

    if not yes:
        click.confirm('Are you sure you want to update {}?'.format(cluster_name), abort=True)

    try:
        generate_terraform_config(paths['terraformTemplatePath'], output_dir, cluster_config)
        terraform_state_path = '{}/terraform.tfstate'.format(output_dir)
        terraform_apply(output_dir, terraform_state_path)

        kops_config_path = generate_kops_config(cluster_config, paths['kopsTemplatePath'], terraform_state_path, output_dir)

        kops_replace_cluster(kops_config_path, cluster_config['ConfigBaseURL'])
        kops_update_cluster(cluster_name, cluster_config['ConfigBaseURL'])

        if yes or click.confirm('Should I initiate a rolling update?'):
            kops_rolling_update(cluster_name, cluster_config['ConfigBaseURL'])
    except Exception as e:
        print("Update cluster failed:", str(e), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__": cli(obj={})

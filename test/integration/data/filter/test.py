import os

def execute_command(filename):
    cmd = "cat " + filename
    os.system(cmd)

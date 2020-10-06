#!/bin/bash

# constants for path to error package
pkg_path="util/errors"
import_path="apic_agents_sdk/pkg/"${pkg_path}

# The return code
RC=0

# These globals are used to track files, names and codes found in the repo
error_files=()
error_names=()
error_codes=()
regular_errors=()
format_errors=()

check_error_line() {
  local line=$1
  local newf=$2

  line=`echo $line` # strips the line of excess spaces
  name=`echo ${line} | awk -F= '{print $1}'` #gets the name of the error
  code=`echo ${line} | awk -F'\\\(' '{print $2}' | awk -F, '{print $1}'` # gets the error code
  err_str=`echo ${line} | awk -F'(New|Newf)\\\(' '{print $2}' | awk -F, '{print $2}' | awk -F'"\\\)' '{print $1}'`\" # gets the error message

  error_names=`echo $error_names && echo $name`
  error_codes=`echo $error_codes && echo $code`

  if [ -z "$newf" ]; then
    regular_errors=`echo $regular_errors && echo $name`
    if grep -q "%s\|%d\|%v"  <<< $err_str; then
      RC=1
      echo Error $name "($file)" uses string format variables but was not declared with Newf
    fi
  else
    format_errors=`echo $format_errors && echo $name`
    if ! grep -q "%s\|%d\|%v"  <<< $err_str; then
      RC=1
      echo Error $name "($file)" is a format error without any format variables
    fi
  fi
}

check_file_for_error_lines() {
  local pattern=$1
  local filename=$2
  local newf=$3
  
  # find any lines that call the New and Newf constructors
  while read line;
  do
    check_error_line "$line" $newf
  done < <(grep $pattern $filename | grep -v "func New")
}

find_errors() {
  # Find all defined errors
  for file in "${go_files[@]}"; do
    err_pkg=""
    # different processing for errors in the errors package
    if grep -q ${pkg_path} <<<"$file"; then  
      err_pkg="true"
      # find uses in the error package itself
      has_pkg=`grep "= New" $file`
    else
      # find modules that import the errors package
      has_pkg=`grep $import_path $file`
    fi

    # check if the grep from has_pkg found matches
    if [ $? -eq 0 ]; then
      # Add the file to teh error_files array
      error_files=`echo $error_files && echo $file`

      # figure out the pakage alias, if any
      errors_pkg_name=`echo $has_pkg | sed -e "s/^import//" | awk '{print $1}' | grep -v $import_path | grep -v import`.
      if [ -z "$errors_pkg_name" ]; then
        errors_pkg_name="errors".
      fi

      # this file is in the error package
      if [ ! -z "$err_pkg" ]; then
        errors_pkg_name=""
      fi

echo ${errors_pkg_name}
echo $file
      check_file_for_error_lines "${errors_pkg_name}New(" $file ""
      check_file_for_error_lines "${errors_pkg_name}Newf(" $file "true"
    fi
  done

  IFS=', ' read -r -a error_names <<< "$error_names"
  IFS=', ' read -r -a error_codes <<< "$error_codes"
  IFS=', ' read -r -a regular_errors <<< "$regular_errors"
  IFS=', ' read -r -a format_errors <<< "$format_errors"
}

find_usages() {
  for file in "${go_files[@]}"; do
    # check for regular errors
    for err in "${regular_errors[@]}"; do
      # find lines with the error in the file
      while read num line; do
        # check if the line calls FormatError
        if grep -q "FormatError(" <<<$line; then
          RC=1
          num=`echo ${num} | awk -F: '{print $1}'`
          echo Error $err used at ${file} \(line ${num}\) and calls FormatError
        fi
      done < <(grep "$err(,|.| )" $file)
    done

    for err in "${format_errors[@]}"; do
      # find lines with the error in the file
      while read num line; do
        # check if the line calls FormatError
        if ! grep -q "FormatError(" <<<$line; then
          RC=1
          num=`echo ${num} | awk -F: '{print $1}'`
          echo Error $err used at ${file} \(line ${num}\) and does not call FormatError
        fi
      done < <(grep -n "$err(,|.| )" $file | grep -v "New(" | grep -v "Newf(")
    done
  done
}

check_for_dependency_errors() {
  local search_dir=$1
  local start_dir=$2
  local dep_dir=$3

  # get errors in dependency
  cd $start_dir
  cd $dep_dir
  go_files=(`find . -name "*.go" -type f | grep -v _test.go`)
  find_errors

  # check for usage of dependency errors
  cd $start_dir
  cd $search_dir
  # find . -name "*.go" -type f
  go_files=(`find . -name "*.go" -type f | grep -v _test.go`)
  find_usages

  # reset errors
  error_files=()
  error_names=()
  error_codes=()
  regular_errors=()
  format_errors=()
}

check_errors_markdown() {
  for err in "${error_names[@]}"; do
    # check the errors are in the markdown
    if ! grep -q "$err" ./errors.md; then
      RC=1
      echo Error $err was not in the errors.md file
    fi
  done

  for code in "${error_codes[@]}"; do
    # check the errors are in the markdown
    if ! grep -q "$code" ./errors.md; then
      RC=1
      echo Error code $code was not in the errors.md file
    fi
  done
}

main() {
  search_dir=$1
  start_dir=`pwd`
  # find errors from dependencies
  for dep_dir in "${@:2}"
  do
    check_for_dependency_errors $search_dir $start_dir $dep_dir
  done

  # move to the search directory
  cd $start_dir
  cd $search_dir

  # Get all go files within the directory
  go_files=(`find . -name "*.go" -type f | grep -v _test.go`)
  find_errors
  find_usages

  # check the markdown file
  cd $start_dir
  check_errors_markdown
}

main $@

exit $RC
#!/env/Python3.10.4
#/MobCat (2024)

"""
FSH24 - Fast Sample Hash 24-byte
Super fast integrity hash using strategic 4MB sampling

Optimized sample size based on benchmarking results
4MB is optimal for most storage systems (NTFS cluster alignment, SSD blocks, etc.)
So the cpu only has to spend one cycle to read one block for hashing.
OPTIMAL_SAMPLE_SIZE = 4194304  # 4 MB (2^22)
"""

import os
import sys
import math
import time
import hashlib
import argparse
import json
from pathlib import Path
import glob

def calculate_optimal_chunks(file_size, sample_size=4194304, target_coverage=0.01):
    """
    Calculate optimal number of middle chunks based on file size
    
    Strategy:
    - Small files (<100MB): 4 total chunks (2 middle) - fixed for speed
    - Medium+ files (100MB+): Calculate chunks to achieve AT LEAST target coverage (default 1%)
    
    Total chunks = first + middle + last
    Returns middle chunk count only
    """
    # Convert bytes to MB
    file_size_mb = file_size / (1024 * 1024)
    
    # Small files: use fixed 4 chunks (2 middle) for speed
    if file_size_mb < 100:
        return 2  # Changed from 1 to 2 for 4 total chunks
    
    # Medium+ files: calculate chunks needed for AT LEAST target coverage
    # Total coverage = total_chunks * sample_size / file_size
    # We want: total_chunks * sample_size / file_size >= target_coverage
    # So: total_chunks >= (target_coverage * file_size) / sample_size
    target_total_chunks = (target_coverage * file_size) / sample_size
    
    # Round UP to ensure we meet at least the target coverage
    target_total_chunks = math.ceil(target_total_chunks)
    
    # Ensure we have at least 4 total chunks (2 middle)
    target_total_chunks = max(4, target_total_chunks)
    
    # Convert total chunks to middle chunks (subtract first and last)
    middle_chunks = target_total_chunks - 2
    
    # Ensure middle chunks is at least 2 (for minimum 4 total chunks)
    middle_chunks = max(2, middle_chunks)  # Changed from 1 to 2
    
    return middle_chunks


def fast_sample_hash(filepath, target_coverage=0.01):
    """
    Super fast integrity hash using strategic 4MB sampling
    Hashes: first chunk + N middle chunks + last chunk + file size
    24 bytes = 48 hex chars
    
    target_coverage: target percentage of file to sample (0.01 = 1%)
    """
    file_size = os.path.getsize(filepath)
    middle_chunks = calculate_optimal_chunks(file_size, 4194304, target_coverage)
    
    hasher = hashlib.blake2b(digest_size=24)
    
    with open(filepath, "rb") as f:
        # Hash first chunk
        first_chunk = f.read(4194304)
        hasher.update(first_chunk)
        
        # Hash multiple middle chunks for better coverage
        if file_size > 4194304 * (middle_chunks + 2):
            for i in range(middle_chunks):
                # Distribute middle chunks evenly across the file
                position = file_size * (i + 2) // (middle_chunks + 2)
                f.seek(position)
                middle_chunk = f.read(4194304)
                hasher.update(middle_chunk)
        
        # Hash last chunk (avoid overlap with middle chunks)
        if file_size > 4194304 * (middle_chunks + 2):
            f.seek(max(0, file_size - 4194304))
            last_chunk = f.read()
            hasher.update(last_chunk)
    
    # Include file size in hash for extra integrity
    hasher.update(file_size.to_bytes(8, 'big'))
    
    return hasher.hexdigest().upper(), middle_chunks + 2


def expand_file_paths(input_paths, recursive=False):
    """
    Expand file paths to handle both files and folders
    Returns a list of file paths with their original relative structure preserved
    """
    expanded_files = []
    
    for input_path in input_paths:
        if os.path.isfile(input_path):
            # It's a file, add it directly
            expanded_files.append(input_path)
        elif os.path.isdir(input_path):
            # It's a directory, get all files from it
            if recursive:
                # Use glob for recursive search
                pattern = os.path.join(input_path, '**', '*')
                files = glob.glob(pattern, recursive=True)
                # Filter out directories, keep only files
                files = [f for f in files if os.path.isfile(f)]
            else:
                # Use glob for non-recursive search
                pattern = os.path.join(input_path, '*')
                files = glob.glob(pattern)
                # Filter out directories, keep only files
                files = [f for f in files if os.path.isfile(f)]
            
            # Sort files for consistent ordering
            files.sort()
            expanded_files.extend(files)
        else:
            # Path doesn't exist, warn but continue
            print(f"Warning: Path not found: {input_path}")
    
    return expanded_files


def process_single_file(filepath, verbose=False, json_output=False, target_coverage=0.01):
    """
    Process a single file and return results
    """
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"File not found: {filepath}")
        
    file_size = os.path.getsize(filepath)
    filename = os.path.basename(filepath)

    if (json_output==False):
        print(f"Processing: {filename}")
    
    start_time = time.time()
    hash_hex, chunks = fast_sample_hash(filepath, target_coverage)
    elapsed_time = time.time() - start_time
    
    coverage_percent = (chunks * 4194304 / file_size) * 100 if file_size > 0 else 0
    
    result = {
        'filename': filename,
        'filepath': filepath,
        'file_size': file_size,
        'fsh24': hash_hex,
        'chunks': chunks,
        'coverage_percent': coverage_percent,
        'processing_time': elapsed_time
    }
    
    if json_output:
        return result
    
    # Console output
    if verbose:
        print(f"File size: {file_size:,} bytes ({result['file_size']/1024/1024:.1f} MB)" if result['file_size'] < 1024**3 else f"File size: {file_size:,} bytes ({result['file_size']/1024/1024/1024:.1f} GB)")
        print(f"FSH24: {hash_hex}")
        print(f"Chunks: {chunks}, Coverage: {coverage_percent:.4f}%, Time: {elapsed_time:.3f}s")
    else:
        print(f"FSH24: {hash_hex}")
    
    return result


def generate_hash_file_multiple(filepaths, output_filename, target_coverage=0.01):
    """
    Generate a hash file in FSH24 format for multiple files
    """
    with open(output_filename, "w") as f:
        f.write("FSH24-1\n")
        
        for filepath in filepaths:
            if not os.path.exists(filepath):
                print(f"Warning: Skipping missing file: {filepath}")
                continue
                
            file_size = os.path.getsize(filepath)
            hash_hex, chunks = fast_sample_hash(filepath, target_coverage)
            f.write(f"{hash_hex}|{chunks}|{file_size}|{filepath}\n")


def verify_hash_file(hash_filename, verbose=False, json_output=False):
    """
    Verify files against a hash file
    """
    if not os.path.exists(hash_filename):
        raise FileNotFoundError(f"Hash file not found: {hash_filename}")
    
    with open(hash_filename, "r") as f:
        lines = f.readlines()
    
    if not lines or not lines[0].strip().startswith("FSH24"):
        raise ValueError("Invalid checksum file. This file is not a FSH24 checksum v1 file.")
    
    results = []
    verified = 0
    failed = 0
    totalSize = 0
    totalHashedSize = 0
    TotalHashedPercentage = 0
    
    # Start timing
    start_time = time.time()
    
    for line in lines[1:]:  # Skip header
        line = line.strip()
        if not line:
            continue
            
        parts = line.split("|")
        if len(parts) != 4:
            if not json_output:
                print(f"Invalid line format: {line}")
            continue
            
        expected_hash, chunks, file_size, filepath = parts
        chunks = int(chunks)
        file_size = int(file_size)
        
        result = {
            'filepath': filepath,
            'filename': os.path.basename(filepath),
            'expected_hash': expected_hash,
            'expected_size': file_size,
            'status': 'unknown'
        }
        
        if not os.path.exists(filepath):
            result['status'] = 'missing'
            if not json_output:
                print(f"!MISSING: {filepath}")
            failed += 1
        else:
            current_size = os.path.getsize(filepath)
            result['actual_size'] = current_size
            
            # Add to total size regardless of verification outcome
            totalSize += current_size
            
            if current_size != file_size:
                result['status'] = 'size_mismatch'
                if not json_output:
                    print(f"!SIZE MISMATCH: {filepath} (expected: {file_size}, actual: {current_size})")
                failed += 1
            else:
                # Show "Checking..." message in verbose mode
                if verbose and not json_output:
                    print(f"{expected_hash}|{chunks}|{file_size}|{filepath}| Checking...", end="", flush=True)
                
                # Time individual file hashing
                file_start_time = time.time()
                current_hash, _ = fast_sample_hash(filepath)
                file_time = time.time() - file_start_time
                
                # Calculate hashed size for this file (chunks * 4MB)
                hashed_size = chunks * 4194304  # 4MB per chunk
                totalHashedSize += hashed_size
                
                result['actual_hash'] = current_hash
                result['processing_time'] = file_time
                result['hashed_size'] = hashed_size
                
                if current_hash != expected_hash:
                    result['status'] = 'hash_mismatch'
                    if not json_output:
                        if verbose:
                            # Overwrite the "Checking..." line
                            print(f"\r{expected_hash}|{chunks}|{file_size}|{filepath}| HASH MISMATCH ✗")
                        else:
                            print(f"HASH MISMATCH: {filepath}")
                    failed += 1
                else:
                    result['status'] = 'verified'
                    if verbose and not json_output:
                        # Overwrite the "Checking..." line
                        print(f"\r{expected_hash}|{chunks}|{file_size}|{filepath}| Verified ✓ ")
                    verified += 1
        
        results.append(result)
    
    # Calculate total time and percentage
    total_time = time.time() - start_time
    TotalHashedPercentage = (totalHashedSize / totalSize * 100) if totalSize > 0 else 0
    
    summary = {
        'verified': verified,
        'failed': failed,
        'total': verified + failed,
        'success': failed == 0,
        'total_time': total_time,
        'average_time_per_file': total_time / (verified + failed) if (verified + failed) > 0 else 0,
        'total_size': totalSize,
        'total_hashed_size': totalHashedSize,
        'total_hashed_percentage': TotalHashedPercentage
    }
    
    if json_output:
        return {
            'summary': summary,
            'results': results
        }
    
    if verbose:
        print(f"\nVerification complete: {verified} verified, {failed} failed")
        print(f"Total time: {total_time:.3f}s")
        if (verified + failed) > 0:
            print(f"Average time per file: {total_time/(verified + failed):.3f}s")
        print(f"Total file size: {totalSize:,} bytes ({totalSize/(1024**3):.2f} GB)")
        print(f"Total hashed size: {totalHashedSize:,} bytes ({totalHashedSize/(1024**3):.2f} GB)")
        print(f"Total hashed percentage: {TotalHashedPercentage:.4f}%")
    else:
        print(f"Verification: {verified} verified, {failed} failed")
    
    return summary


def main():
    parser = argparse.ArgumentParser(
        description="FSH24 - Fast Sample Hash 24-byte integrity checker\nAims to make checking the integrity of 40gb game files after you download them easy.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  fsh24.py file.ext                              # Basic hash (single file)
  fsh24.py "file1.ext" "../folder/file2.ext" "C:/path/file3.ext" # Multiple files
  fsh24.py folder/                               # Hash all files in folder
  fsh24.py folder/ -r                            # Hash all files in folder recursively
  fsh24.py file.ext -v                           # Verbose hash
  fsh24.py file.ext -o output.fsh24              # Custom output file
  fsh24.py file.ext -j                           # JSON output
  fsh24.py checksums.fsh24                       # Verify hash file
  fsh24.py checksums.fsh24 -v                    # Verbose verify
        """
    )
    
    parser.add_argument('files', nargs='+', help='Input file(s), folder(s), or .fsh24 hash file to verify')
    parser.add_argument('-o', '--output', help='Output .fsh24 file name (default: checksums.fsh24)')
    parser.add_argument('-v', '--verbose', action='store_true', help='Verbose output')
    parser.add_argument('-j', '--json', action='store_true', help='JSON output')
    parser.add_argument('-r', '--recursive', action='store_true', help='Recursively process folders')
    
    args = parser.parse_args()
    
    try:
        # Check if we have a single .fsh24 file (verify mode)
        if len(args.files) == 1 and args.files[0].lower().endswith('.fsh24'):
            # Verify mode
            if args.json:
                result = verify_hash_file(args.files[0], args.verbose, json_output=True)
                print(json.dumps(result, indent=2))
            else:
                verify_hash_file(args.files[0], args.verbose)
                input("\nPress Enter to exit...")
        else:
            # Hash mode (files and/or folders)
            # Expand all input paths to get actual files
            expanded_files = expand_file_paths(args.files, recursive=args.recursive)
            
            if not expanded_files:
                print("No files found to process.")
                sys.exit(1)
            
            if args.json:
                # Process all files and collect results
                results = []
                total_start = time.time()
                
                for filepath in expanded_files:
                    if not os.path.exists(filepath):
                        print(f"Warning: Skipping missing file: {filepath}")
                        continue
                    
                    result = process_single_file(filepath, args.verbose, json_output=True, target_coverage=0.01)
                    results.append(result)
                
                total_time = time.time() - total_start
                
                output_data = {
                    'magic': 'FSH24-1',
                    'total_files': len(results),
                    'total_processing_time': total_time,
                    'average_time_per_file': total_time / len(results) if results else 0,
                    'files': results
                }
                
                if args.output:
                    # Save JSON to file
                    with open(args.output, 'w') as f:
                        json.dump(output_data, f, indent=2)
                    print(f"JSON saved to: {args.output}")
                else:
                    print(json.dumps(output_data, indent=2))
            else:
                # Process files with console output
                processed_files = []
                total_start = time.time()
                
                for filepath in expanded_files:
                    if not os.path.exists(filepath):
                        print(f"Warning: Skipping missing file: {filepath}")
                        continue
                    
                    process_single_file(filepath, args.verbose, target_coverage=0.01)
                    processed_files.append(filepath)
                    
                    if len(expanded_files) > 1:  # Add separator for multiple files
                        print()
                
                total_time = time.time() - total_start
                
                if processed_files:
                    # Generate hash file
                    output_file = args.output if args.output else "checksums.fsh24"
                    generate_hash_file_multiple(processed_files, output_file, 0.01)
                    
                    if len(processed_files) > 1:
                        # Calculate totals for enhanced summary
                        total_file_size = 0
                        total_hashed_size = 0
                        
                        for filepath in processed_files:
                            file_size = os.path.getsize(filepath)
                            middle_chunks = calculate_optimal_chunks(file_size, 4194304, 0.01)
                            chunks = middle_chunks + 2
                            hashed_size = chunks * 4194304  # 4MB per chunk
                            
                            total_file_size += file_size
                            total_hashed_size += hashed_size
                        
                        # Calculate percentages
                        total_hash_percentage = (total_hashed_size / total_file_size * 100) if total_file_size > 0 else 0
                        
                        print(f"Processed {len(processed_files)} files in {total_time:.3f}s")
                        print(f"Total file size: {total_file_size:,} bytes ({total_file_size/(1024**3):.2f} GB)")
                        print(f"Total hashed size: {total_hashed_size:,} bytes ({total_hashed_size/(1024**3):.2f} GB)")
                        print(f"Total hash percentage: {total_hash_percentage:.4f}%")
                    
                    if not args.verbose:
                        print(f"Hash file saved: {output_file}")
                    
                    input("\nPress Enter to exit...")
                
    except FileNotFoundError as e:
        print(f"Error: {e}")
        sys.exit(1)
    except ValueError as e:
        print(f"Error: {e}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nOperation cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
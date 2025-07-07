#!/env/Python3.10.4
#/MobCat (2024)


"""
File Corruption Testing Script

This script creates corrupted versions of files to simulate:
1. Partially corrupted files (scrambled data)
2. Incomplete downloads (truncated files)

Uses streaming for large files and shows progress bars.

Usage: python corrupt.py <input_file>
"""

import os
import sys
import random
import argparse
from pathlib import Path
from tqdm import tqdm


def get_file_size(file_path):
    """Get file size in bytes."""
    return os.path.getsize(file_path)


def stream_corrupt_file(input_file, output_file, corruption_percentage, file_size, 
                       progress_bar=None, chunk_size=8192):
    """
    Stream and corrupt a file chunk by chunk.
    
    Args:
        input_file: input file path
        output_file: output file path
        corruption_percentage: percentage of data to corrupt (0-100)
        file_size: total file size for progress calculation
        progress_bar: tqdm progress bar object
        chunk_size: size of chunks to read at once
    """
    bytes_processed = 0
    
    with open(input_file, 'rb') as infile, open(output_file, 'wb') as outfile:
        while True:
            chunk = infile.read(chunk_size)
            if not chunk:
                break
            
            # Corrupt the chunk if needed
            if corruption_percentage > 0:
                chunk = corrupt_chunk(chunk, corruption_percentage)
            
            outfile.write(chunk)
            bytes_processed += len(chunk)
            
            # Update progress bar
            if progress_bar:
                progress_bar.update(len(chunk))


def stream_truncate_file(input_file, output_file, percentage_to_keep, file_size,
                        progress_bar=None, chunk_size=8192):
    """
    Stream and truncate a file to simulate incomplete download.
    
    Args:
        input_file: input file path
        output_file: output file path
        percentage_to_keep: percentage of data to keep (0-100)
        file_size: total file size
        progress_bar: tqdm progress bar object
        chunk_size: size of chunks to read at once
    """
    bytes_to_keep = int(file_size * percentage_to_keep / 100)
    bytes_processed = 0
    
    with open(input_file, 'rb') as infile, open(output_file, 'wb') as outfile:
        while bytes_processed < bytes_to_keep:
            # Calculate how much to read this iteration
            remaining_bytes = bytes_to_keep - bytes_processed
            read_size = min(chunk_size, remaining_bytes)
            
            chunk = infile.read(read_size)
            if not chunk:
                break
            
            outfile.write(chunk)
            bytes_processed += len(chunk)
            
            # Update progress bar
            if progress_bar:
                progress_bar.update(len(chunk))


def corrupt_chunk(chunk, corruption_percentage):
    """
    Corrupt a percentage of bytes in a chunk.
    
    Args:
        chunk: bytes object to corrupt
        corruption_percentage: percentage of bytes to corrupt (0-100)
    
    Returns:
        bytes: corrupted chunk
    """
    if corruption_percentage <= 0:
        return chunk
    
    chunk_list = bytearray(chunk)
    total_bytes = len(chunk_list)
    bytes_to_corrupt = int(total_bytes * corruption_percentage / 100)
    
    if bytes_to_corrupt > 0:
        # Randomly select positions to corrupt
        positions_to_corrupt = random.sample(range(total_bytes), 
                                           min(bytes_to_corrupt, total_bytes))
        
        # Replace selected bytes with random values
        for pos in positions_to_corrupt:
            chunk_list[pos] = random.randint(0, 255)
    
    return bytes(chunk_list)


def create_corruption_samples(input_file, output_dir, corruption_levels, samples_per_level=3):
    """
    Create corrupted file samples with specified corruption levels using streaming.
    
    Args:
        input_file: path to input file
        output_dir: directory to save corrupted files
        corruption_levels: list of corruption percentages
        samples_per_level: number of samples to create per corruption level
    """
    input_path = Path(input_file)
    base_name = input_path.stem
    extension = input_path.suffix
    file_size = get_file_size(input_file)
    
    print(f"Creating corruption samples for {input_file}")
    print(f"Original file size: {file_size:,} bytes ({file_size / (1024**3):.2f} GB)")
    
    # Calculate total operations for overall progress
    total_operations = len(corruption_levels) * samples_per_level
    
    with tqdm(total=total_operations, desc="Overall Progress", unit="file", position=0) as overall_pbar:
        for corruption_level in corruption_levels:
            print(f"\nCreating {samples_per_level} samples with {corruption_level}% corruption...")
            
            for sample_num in range(1, samples_per_level + 1):
                # Set seed for reproducible corruption patterns
                random.seed(corruption_level * 1000 + sample_num)
                
                output_filename = f"{base_name}-{corruption_level}-{sample_num}{extension}"
                output_path = os.path.join(output_dir, output_filename)
                
                # Create progress bar for current file
                with tqdm(total=file_size, desc=f"Creating {output_filename}", 
                         unit="B", unit_scale=True, position=1, leave=False) as file_pbar:
                    
                    stream_corrupt_file(input_file, output_path, corruption_level, 
                                      file_size, file_pbar)
                
                print(f"  ✓ Created: {output_filename}")
                overall_pbar.update(1)


def create_incomplete_downloads(input_file, output_dir, completion_levels):
    """
    Create incomplete download samples with specified completion levels using streaming.
    
    Args:
        input_file: path to input file
        output_dir: directory to save incomplete files
        completion_levels: list of completion percentages
    """
    input_path = Path(input_file)
    base_name = input_path.stem
    extension = input_path.suffix
    file_size = get_file_size(input_file)
    
    print(f"\nCreating incomplete download samples for {input_file}")
    print(f"Original file size: {file_size:,} bytes ({file_size / (1024**3):.2f} GB)")
    
    # Calculate total operations for overall progress
    total_operations = len(completion_levels)
    
    with tqdm(total=total_operations, desc="Overall Progress", unit="file", position=0) as overall_pbar:
        for completion_level in completion_levels:
            output_filename = f"{base_name}-incomplete-{completion_level}{extension}"
            output_path = os.path.join(output_dir, output_filename)
            
            expected_size = int(file_size * completion_level / 100)
            
            # Create progress bar for current file
            with tqdm(total=expected_size, desc=f"Creating {output_filename}", 
                     unit="B", unit_scale=True, position=1, leave=False) as file_pbar:
                
                stream_truncate_file(input_file, output_path, completion_level, 
                                   file_size, file_pbar)
            
            actual_size = get_file_size(output_path)
            print(f"  ✓ Created: {output_filename} ({actual_size:,} bytes, {completion_level}% complete)")
            overall_pbar.update(1)


def main():
    parser = argparse.ArgumentParser(description='Create corrupted file samples for testing')
    parser.add_argument('input_file', help='Input file to corrupt')
    parser.add_argument('--samples-per-level', '-s', type=int, default=3,
                       help='Number of samples to create per corruption level (default: 3)')
    parser.add_argument('--corruption-only', action='store_true',
                       help='Only create corruption samples, not incomplete downloads')
    parser.add_argument('--incomplete-only', action='store_true',
                       help='Only create incomplete download samples, not corruption samples')
    parser.add_argument('--chunk-size', type=int, default=8192,
                       help='Chunk size for streaming in bytes (default: 8192)')
    
    args = parser.parse_args()
    
    # Validate input file
    if not os.path.exists(args.input_file):
        print(f"Error: Input file '{args.input_file}' not found")
        sys.exit(1)
    
    # Set output directory to 'out'
    output_dir = 'out'
    
    # Create output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)
    print(f"Output directory: {os.path.abspath(output_dir)}")
    
    # Validate samples per level
    if args.samples_per_level < 1:
        print("Error: samples-per-level must be at least 1")
        sys.exit(1)
    
    # Define corruption levels and samples
    corruption_levels = [10, 20, 35, 40, 50, 60, 75, 80, 90, 100]
    
    # Define incomplete download levels (90-99% completion)
    incomplete_levels = list(range(90, 100))
    
    try:
        print(f"Using {args.samples_per_level} samples per corruption level")
        print(f"Chunk size: {args.chunk_size:,} bytes")
        print("=" * 60)
        
        if not args.incomplete_only:
            create_corruption_samples(args.input_file, output_dir, 
                                    corruption_levels, args.samples_per_level)
        
        if not args.corruption_only:
            create_incomplete_downloads(args.input_file, output_dir, 
                                      incomplete_levels)
        
        print("\n" + "=" * 60)
        print(f"✓ All test files created successfully in: {os.path.abspath(output_dir)}")
        
        # Show summary
        total_corruption_files = len(corruption_levels) * args.samples_per_level if not args.incomplete_only else 0
        total_incomplete_files = len(incomplete_levels) if not args.corruption_only else 0
        total_files = total_corruption_files + total_incomplete_files
        
        print(f"Summary:")
        if not args.incomplete_only:
            print(f"  - Corruption samples: {total_corruption_files} files")
        if not args.corruption_only:
            print(f"  - Incomplete downloads: {total_incomplete_files} files")
        print(f"  - Total files created: {total_files}")
        
    except KeyboardInterrupt:
        print("\n\nOperation cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
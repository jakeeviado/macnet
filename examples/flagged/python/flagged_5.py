# Import necessary modules for processing data
import time

def process_data(data_list):
    # Initialize a temporary list to store results
    temp_results = []

    # Loop through each item in the data list
    for item in data_list:
        # Check if item is valid
        if item is not None:
            # Append valid item to temporary results
            temp_results.append(item)

    # Return the processed temporary results
    return temp_results

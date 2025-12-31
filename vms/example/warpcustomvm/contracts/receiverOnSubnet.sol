// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// SPDX-License-Identifier: Ecosystem

pragma solidity ^0.8.18;

import "@teleporter/ITeleporterMessenger.sol";
import "@teleporter/ITeleporterReceiver.sol";


contract ReceiverOnSubnet is ITeleporterReceiver {
   ITeleporterMessenger public immutable messenger = ITeleporterMessenger(0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf);   
  
    // ERC20 Token Storage
    string public constant name = "Custom L1 Token";
    string public constant symbol = "CL1T";
    uint8 public constant decimals = 18;
    uint256 public totalSupply;
    
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    
    // Hardcoded recipient address for minting
    address public constant MINT_RECIPIENT = 0x0eBC9Aa0f45A16A9D68c10D6eE81eD3084aCeaf3;
    
    // Amount to mint per message received (1 token = 1e18)
    uint256 public constant MINT_AMOUNT = 1 ether;
    
    // Amount to burn per message received (1 token = 1e18)
    uint256 public constant BURN_AMOUNT = 1 ether;

    string public lastMessage;
    string public lastError;

    // ERC20 Events
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event MessageReceivedAndMinted(string message, address recipient, uint256 amount);
    event MessageReceivedAndBurned(string message, address from, uint256 amount);
    event MessageSendFailed(string reason);
    
    function receiveTeleporterMessage(bytes32, address, bytes calldata message) external {
        // Only the Teleporter receiver can deliver a message.
        require(msg.sender == address(messenger), "CustomL1ToCChain: unauthorized TeleporterMessenger");

        // Decode and store the message first
        string memory decodedMessage = abi.decode(message, (string));
        lastMessage = decodedMessage;
        
        // Only mint if the message is "mint"
        if (keccak256(abi.encodePacked(decodedMessage)) == keccak256(abi.encodePacked("mint"))) {
            // Mint tokens to the hardcoded address FIRST (state change)
            _mint(MINT_RECIPIENT, MINT_AMOUNT);
            
            // Create dynamic confirmation message
            string memory confirmationMessage = string(abi.encodePacked(
                "Minted ", 
                _toString(MINT_AMOUNT / 1 ether), 
                " token(s) to ", 
                _toHexString(MINT_RECIPIENT),
                ". New total supply: ",
                _toString(totalSupply / 1 ether)
            ));
            
            // Update lastMessage
            lastMessage = confirmationMessage;
            
            emit MessageReceivedAndMinted(confirmationMessage, MINT_RECIPIENT, MINT_AMOUNT);
            
            // Try to send confirmation back to custom VM (don't revert if this fails)
            try messenger.sendCrossChainMessage(
                TeleporterMessageInput({
                    destinationBlockchainID: 0xa29f2cd4246047db8dd8dfd4b189abb0f4a68698767ebabe209e9d1c059dc6a1,
                    destinationAddress: address(0x1111111111111111111111111111111111111111),
                    feeInfo: TeleporterFeeInfo({feeTokenAddress: address(0), amount: 0}),
                    requiredGasLimit: 100000,
                    allowedRelayerAddresses: new address[](0),
                    message: abi.encode(confirmationMessage)
                })
            ) {
                // Success - message sent
            } catch Error(string memory reason) {
                // Failed - but minting still succeeded
                lastError = reason;
                emit MessageSendFailed(reason);
            } catch {
                lastError = "Unknown error sending message";
                emit MessageSendFailed("Unknown error");
            }
        }
        // Only burn if the message is "burn"
        else if (keccak256(abi.encodePacked(decodedMessage)) == keccak256(abi.encodePacked("burn"))) {
            // Burn tokens from the hardcoded address FIRST (state change)
            _burn(MINT_RECIPIENT, BURN_AMOUNT);
            
            // Create dynamic confirmation message
            string memory confirmationMessage = string(abi.encodePacked(
                "Burned ", 
                _toString(BURN_AMOUNT / 1 ether), 
                " token(s) from ", 
                _toHexString(MINT_RECIPIENT),
                ". Remaining total supply: ",
                _toString(totalSupply / 1 ether)
            ));
            
            // Update lastMessage
            lastMessage = confirmationMessage;
            
            emit MessageReceivedAndBurned(confirmationMessage, MINT_RECIPIENT, BURN_AMOUNT);
            
            // Try to send confirmation back to custom VM (don't revert if this fails)
            try messenger.sendCrossChainMessage(
                TeleporterMessageInput({
                    destinationBlockchainID: 0xa29f2cd4246047db8dd8dfd4b189abb0f4a68698767ebabe209e9d1c059dc6a1,
                    destinationAddress: address(0x1111111111111111111111111111111111111111),
                    feeInfo: TeleporterFeeInfo({feeTokenAddress: address(0), amount: 0}),
                    requiredGasLimit: 100000,
                    allowedRelayerAddresses: new address[](0),
                    message: abi.encode(confirmationMessage)
                })
            ) {
                // Success - message sent
            } catch Error(string memory reason) {
                // Failed - but burning still succeeded
                lastError = reason;
                emit MessageSendFailed(reason);
            } catch {
                lastError = "Unknown error sending message";
                emit MessageSendFailed("Unknown error");
            }
        }
        else {
            // For any other message, just store it
            lastMessage = decodedMessage;
        }
    }
    
    // ERC20 Functions
    function transfer(address to, uint256 amount) external returns (bool) {
        require(to != address(0), "ERC20: transfer to zero address");
        require(balanceOf[msg.sender] >= amount, "ERC20: insufficient balance");
        
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        
        emit Transfer(msg.sender, to, amount);
        return true;
    }
    
    function approve(address spender, uint256 amount) external returns (bool) {
        require(spender != address(0), "ERC20: approve to zero address");
        
        allowance[msg.sender][spender] = amount;
        
        emit Approval(msg.sender, spender, amount);
        return true;
    }
    
    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        require(from != address(0), "ERC20: transfer from zero address");
        require(to != address(0), "ERC20: transfer to zero address");
        require(balanceOf[from] >= amount, "ERC20: insufficient balance");
        require(allowance[from][msg.sender] >= amount, "ERC20: insufficient allowance");
        
        balanceOf[from] -= amount;
        balanceOf[to] += amount;
        allowance[from][msg.sender] -= amount;
        
        emit Transfer(from, to, amount);
        return true;
    }
    
    function _mint(address to, uint256 amount) internal {
        require(to != address(0), "ERC20: mint to zero address");
        
        totalSupply += amount;
        balanceOf[to] += amount;
        
        emit Transfer(address(0), to, amount);
    }
    
    function _burn(address from, uint256 amount) internal {
        require(from != address(0), "ERC20: burn from zero address");
        require(balanceOf[from] >= amount, "ERC20: burn amount exceeds balance");
        
        balanceOf[from] -= amount;
        totalSupply -= amount;
        
        emit Transfer(from, address(0), amount);
    }
    
    // Helper function to convert uint256 to string
    function _toString(uint256 value) internal pure returns (string memory) {
        if (value == 0) {
            return "0";
        }
        uint256 temp = value;
        uint256 digits;
        while (temp != 0) {
            digits++;
            temp /= 10;
        }
        bytes memory buffer = new bytes(digits);
        while (value != 0) {
            digits -= 1;
            buffer[digits] = bytes1(uint8(48 + uint256(value % 10)));
            value /= 10;
        }
        return string(buffer);
    }
    
    // Helper function to convert address to hex string
    function _toHexString(address addr) internal pure returns (string memory) {
        bytes memory buffer = new bytes(42);
        buffer[0] = '0';
        buffer[1] = 'x';
        for (uint256 i = 0; i < 20; i++) {
            uint8 value = uint8(uint160(addr) >> (8 * (19 - i)));
            buffer[2 + i * 2] = _toHexChar(value >> 4);
            buffer[3 + i * 2] = _toHexChar(value & 0x0f);
        }
        return string(buffer);
    }
    
    // Helper function to convert a nibble to hex character
    function _toHexChar(uint8 value) internal pure returns (bytes1) {
        if (value < 10) {
            return bytes1(uint8(48 + value)); // 0-9
        } else {
            return bytes1(uint8(87 + value)); // a-f
        }
    }
}
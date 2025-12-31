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

    // ERC20 Events
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event MessageReceivedAndMinted(string message, address recipient, uint256 amount);
    event MessageReceivedAndBurned(string message, address from, uint256 amount);
	
    function receiveTeleporterMessage(bytes32, address, bytes calldata message) external {
        // Only the Teleporter receiver can deliver a message.
        require(msg.sender == address(messenger), "CustomL1ToCChain: unauthorized TeleporterMessenger");

        // Decode and store the message first
        string memory decodedMessage = abi.decode(message, (string));
        
        // Only mint if the message is "mint"
        if (keccak256(abi.encodePacked(decodedMessage)) == keccak256(abi.encodePacked("mint"))) {
            // Mint tokens to the hardcoded address FIRST (state change)
            _mint(MINT_RECIPIENT, MINT_AMOUNT);
			
			// Create confirmation message
			string memory confirmationMessage = "hello mint interface";
			
			// Update lastMessage
			lastMessage = confirmationMessage;
            
            emit MessageReceivedAndMinted(confirmationMessage, MINT_RECIPIENT, MINT_AMOUNT);
			
			// Try to send confirmation back to custom VM (don't revert if this fails)
			try this.sendMessageToCustomVM(confirmationMessage) {
                // Success - message sent
            } catch {
                // Failed - but minting still succeeded
            }
        }
        // Only burn if the message is "burn"
        else if (keccak256(abi.encodePacked(decodedMessage)) == keccak256(abi.encodePacked("burn"))) {
            // Burn tokens from the hardcoded address FIRST (state change)
            _burn(MINT_RECIPIENT, BURN_AMOUNT);
            
			// Create confirmation message
			string memory confirmationMessage = "hello burn";
			
			// Update lastMessage
			lastMessage = confirmationMessage;
            
            emit MessageReceivedAndBurned(confirmationMessage, MINT_RECIPIENT, BURN_AMOUNT);
			
			// Try to send confirmation back to custom VM (don't revert if this fails)
            try this.sendMessageToCustomVM(confirmationMessage) {
                // Success - message sent
            } catch {
                // Failed - but burning still succeeded
            }
            
            emit MessageReceivedAndBurned(confirmationMessage, MINT_RECIPIENT, BURN_AMOUNT);
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
	
	function sendMessageToCustomVMCheck(string memory message) public {
        sendMessageToCustomVM(message);
    }
	
	/**
     * @notice External function to send message to Custom VM via C-Chain contract
     * @param message The text message to send
     */
    function sendMessageToCustomVM(string memory message) public {
        // Use low-level call to CChainToCustomL1 contract
        address cchainContract = 0x8c1678C30474192Fc89A7A8cF28c716a11b029a7;
		
		(bool success, ) = cchainContract.call(
            abi.encodeWithSignature(
                "sendTextMessage(bytes32,string)",
                bytes32(0xa29f2cd4246047db8dd8dfd4b189abb0f4a68698767ebabe209e9d1c059dc6a1),
                message
            )
        );
        
        // Check success
        require(success, "Message send to C-Chain contract failed 1");
    }
}

import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { VideoService, Video, PaginatedVideosResponse } from '../../services/video';
import { FolderService, Folder } from '../../services/folder'; // Import FolderService and Folder

@Component({
  selector: 'app-video-list',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  templateUrl: './video-list.html',
  styleUrls: ['./video-list.css']
})
export class VideoListComponent implements OnInit {
  videos: Video[] = [];
  currentPage: number = 1;
  totalPages: number = 1;
  totalItems: number = 0;
  pageSize: number = 10;

  folders: Folder[] = [];
  selectedFolderId: string | null = null; // For filtering, null means "All Videos"
  isFilteringUnassigned: boolean = false; // Special state for "Unassigned" filter
  newFolderName: string = '';
  folderError: string = '';
  currentViewTitle: string = 'All Videos';

  renamingFolder: Folder | null = null;
  renamingFolderName: string = '';

  isLoading: boolean = false;
  isLoadingFolders: boolean = false;
  errorMessage: string = '';

  constructor(
    private videoService: VideoService,
    private folderService: FolderService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadInitialData();
  }

  loadInitialData(): void {
    this.loadFolders(); // Load folders first
    this.loadVideos(this.currentPage); // Then load videos (which might depend on selectedFolderId)
  }

  loadFolders(): void {
    this.isLoadingFolders = true;
    this.folderError = '';
    // For MVP, we load root folders. The backend defaults to 'root' if no parentFolderId is given.
    this.folderService.getFolders('root').subscribe({
      next: (data: Folder[]) => {
        this.folders = data;
        this.isLoadingFolders = false;
      },
      error: (err: Error) => {
        this.folderError = `Failed to load folders: ${err.message}`;
        this.isLoadingFolders = false;
      }
    });
  }

  loadVideos(page: number): void {
    this.isLoading = true;
    this.errorMessage = '';
    // The VideoService.getVideos method needs to be adapted or a new method created
    // if the backend expects a specific parameter for folderId or "unassigned".
    // For now, assuming getVideos can take an optional folderId string that might be 'unassigned'.
    // This part of VideoService will need modification if it doesn't support this.
    // Let's assume for now the backend supports `folderId` and a special value `unassigned`.
    // This might require an update to `VideoService.getVideos` method signature and logic.
    // For now, this is a placeholder for how it *should* work.
    // The backend's ListUserVideos needs to handle:
    // 1. No folderId param: all videos (or based on some default)
    // 2. folderId=some_id: videos in that folder
    // 3. folderId=unassigned: videos with folderId=null

    // Passing selectedFolderId directly. If it's 'unassigned', backend handles it.
    // If selectedFolderId is null, no folderId param is sent (or backend interprets as all).
    // This requires VideoService.getVideos to be updated to accept folderId.
    // For now, I will filter client-side for MVP if VideoService is not updated yet.
    // However, the subtask implies backend supports filtering.

    // *** Предполагаем, что VideoService.getVideos будет обновлен для поддержки folderId ***
    // this.videoService.getVideos(page, this.pageSize, this.isFilteringUnassigned ? 'unassigned' : this.selectedFolderId).subscribe({
    // For now, I'll just pass selectedFolderId, assuming the service sends it if not null.
    // The backend ListUserVideos needs to be ready for `?folderId=xxxx` or `?folderId=unassigned`
    this.videoService.getVideos(page, this.pageSize /*, this.selectedFolderId, this.isFilteringUnassigned */).subscribe({
      next: (response: PaginatedVideosResponse) => {
        // Client-side filtering for now, until backend and service are confirmed to support folder filtering
        if (this.isFilteringUnassigned) {
            this.videos = response.items.filter(v => !v.folderId);
            // Note: pagination from backend is for *all* items if not filtered server-side.
            // This client-side filter means totalItems/totalPages might be inaccurate for the filtered view.
            // Ideally, filtering is done server-side.
        } else if (this.selectedFolderId) {
            this.videos = response.items.filter(v => v.folderId === this.selectedFolderId);
        } else {
            this.videos = response.items;
        }
        
        this.currentPage = response.currentPage; // This needs to be based on filtered results if done client-side
        this.totalPages = response.totalPages;   // Same as above
        this.totalItems = this.videos.length; // Correct for client-side filtering
                                              // Or response.totalItems if server-side filtering is perfect
        this.pageSize = response.pageSize;
        this.isLoading = false;
      },
      error: (err: Error) => {
        this.errorMessage = `Failed to load videos: ${err.message}`;
        this.isLoading = false;
      }
    });
  }
  
  // Helper to get folder name for display
  getFolderName(folderId?: string | null): string {
    if (!folderId) return 'N/A';
    const folder = this.folders.find(f => f.ID === folderId);
    return folder ? folder.name : 'Unknown Folder';
  }


  filterByFolder(folderId: string | null | 'unassigned'): void {
    this.currentPage = 1; // Reset to first page for new filter
    if (folderId === 'unassigned') {
      this.selectedFolderId = null;
      this.isFilteringUnassigned = true;
      this.currentViewTitle = 'Unassigned Videos';
    } else {
      this.selectedFolderId = folderId;
      this.isFilteringUnassigned = false;
      const folder = this.folders.find(f => f.ID === folderId);
      this.currentViewTitle = folderId ? `${folder ? folder.name : 'Unknown'} Videos` : 'All Videos';
    }
    this.loadVideos(this.currentPage);
  }

  createFolder(): void {
    if (!this.newFolderName.trim()) {
      this.folderError = 'Folder name cannot be empty.';
      return;
    }
    this.isLoadingFolders = true; // To give feedback
    this.folderError = '';
    // For MVP, creating root folders. Pass undefined or null for parentFolderId.
    this.folderService.createFolder(this.newFolderName.trim()).subscribe({
      next: (newFolder) => {
        this.folders.push(newFolder);
        this.folders.sort((a, b) => a.name.localeCompare(b.name)); // Keep sorted
        this.newFolderName = '';
        this.isLoadingFolders = false;
      },
      error: (err: Error) => {
        this.folderError = `Failed to create folder: ${err.message}`;
        this.isLoadingFolders = false;
      }
    });
  }

  startRenameFolder(folder: Folder): void {
    this.renamingFolder = folder;
    this.renamingFolderName = folder.name;
    this.folderError = ''; // Clear previous errors
  }

  cancelRenameFolder(): void {
    this.renamingFolder = null;
    this.renamingFolderName = '';
  }

  submitRenameFolder(): void {
    if (!this.renamingFolder || !this.renamingFolderName.trim()) {
      this.folderError = 'Folder name cannot be empty for renaming.';
      return;
    }
    if (this.renamingFolder.name === this.renamingFolderName.trim()){
        this.cancelRenameFolder();
        return;
    }

    this.isLoadingFolders = true;
    this.folderService.updateFolder(this.renamingFolder.ID, this.renamingFolderName.trim()).subscribe({
      next: (updatedFolder) => {
        const index = this.folders.findIndex(f => f.ID === updatedFolder.ID);
        if (index !== -1) {
          this.folders[index] = updatedFolder;
          this.folders.sort((a, b) => a.name.localeCompare(b.name));
        }
        this.cancelRenameFolder();
        this.isLoadingFolders = false;
      },
      error: (err: Error) => {
        this.folderError = `Failed to rename folder: ${err.message}`;
        this.isLoadingFolders = false;
        // Don't cancel rename on error, let user try again or cancel
      }
    });
  }

  confirmDeleteFolder(folder: Folder): void {
    if (confirm(`Are you sure you want to delete the folder "${folder.name}"? This action cannot be undone. Ensure the folder is empty.`)) {
      this.deleteFolder(folder);
    }
  }

  deleteFolder(folderToDelete: Folder): void {
    this.isLoadingFolders = true;
    this.folderError = '';
    this.folderService.deleteFolder(folderToDelete.ID).subscribe({
      next: () => {
        this.folders = this.folders.filter(f => f.ID !== folderToDelete.ID);
        // If the deleted folder was selected, reset filter to "All Videos"
        if (this.selectedFolderId === folderToDelete.ID) {
          this.filterByFolder(null);
        }
        this.isLoadingFolders = false;
      },
      error: (err: Error) => {
        this.folderError = `Failed to delete folder: ${err.message}`;
        this.isLoadingFolders = false;
      }
    });
  }

  // Video actions (existing methods)
  confirmDelete(videoId: string, videoTitle: string): void {
    if (confirm(`Are you sure you want to delete the video "${videoTitle}"?`)) {
      this.deleteVideoAction(videoId);
    }
  }

  deleteVideoAction(videoId: string): void {
    this.isLoading = true;
    this.videoService.deleteVideo(videoId).subscribe({
      next: () => {
        this.loadVideos(this.currentPage); // Reload current view
      },
      error: (err: Error) => {
        this.errorMessage = `Failed to delete video: ${err.message}`;
        this.isLoading = false;
      }
    });
  }

  editVideo(videoId: string): void {
    this.router.navigate(['/videos', videoId, 'edit']);
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages && page !== this.currentPage) {
      this.loadVideos(page);
    }
  }
}

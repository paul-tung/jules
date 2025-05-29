import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms'; // For ngModel
import { ActivatedRoute, Router, RouterModule } from '@angular/router'; // For route params and navigation
import { Subscription } from 'rxjs';
import { VideoService, Video, UpdateVideoPayload } from '../../services/video'; // Corrected path
import { FolderService, Folder } from '../../services/folder'; // Import FolderService and Folder
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';

@Component({
  selector: 'app-video-edit',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  templateUrl: './video-edit.html',
  styleUrls: ['./video-edit.css']
})
export class VideoEditComponent implements OnInit, OnDestroy {
  video: Video | null = null;
  editableVideo: UpdateVideoPayload = {};
  tagsString: string = '';
  currentVideoTitle: string = '';

  videoPlayerUrl: SafeUrl | string = '';

  availableFolders: Folder[] = [];
  selectedFolderId: string | null = null; // For the folder assignment dropdown

  isLoading: boolean = false;
  isSavingMetadata: boolean = false; // For metadata save
  isAssigningFolder: boolean = false; // For folder assignment action

  errorMessage: string = '';
  successMessage: string = ''; // For metadata save success
  updateErrorMessage: string = ''; // For metadata save error
  folderAssignmentMessage: { type: 'success' | 'error', text: string } = { type: 'success', text: '' };


  private routeSub: Subscription | undefined;
  private videoId: string | null = null;

  constructor(
    private videoService: VideoService,
    private folderService: FolderService, // Inject FolderService
    private route: ActivatedRoute,
    private router: Router,
    private sanitizer: DomSanitizer
  ) {}

  ngOnInit(): void {
    this.loadAvailableFolders(); // Load folders for the dropdown
    this.routeSub = this.route.paramMap.subscribe(params => {
      this.videoId = params.get('videoId');
      if (this.videoId) {
        this.loadVideoDetails(this.videoId);
      } else {
        this.errorMessage = 'Video ID not provided in the route.';
      }
    });
  }

  loadAvailableFolders(): void {
    // For MVP, loading root folders. Could be extended to load all or allow navigation.
    this.folderService.getFolders('root').subscribe({
      next: (folders) => {
        this.availableFolders = folders;
      },
      error: (err) => {
        // Handle error loading folders (e.g., display a message)
        console.error('Failed to load folders:', err);
        this.folderAssignmentMessage = { type: 'error', text: 'Could not load folders for assignment.'};
      }
    });
  }

  loadVideoDetails(id: string): void {
    this.isLoading = true;
    this.errorMessage = '';
    this.successMessage = '';
    this.updateErrorMessage = '';
    this.folderAssignmentMessage.text = '';


    this.videoService.getVideo(id).subscribe({
      next: (data: Video) => {
        this.video = data;
        this.currentVideoTitle = data.title;
        this.editableVideo = {
          title: data.title,
          description: data.description || '',
          tags: data.tags || [], // This is an array, but form binds to tagsString
          status: data.status
        };
        this.tagsString = (data.tags || []).join(', ');
        this.selectedFolderId = data.folderId || null; // Set current folder for dropdown
        this.isLoading = false;

        if (this.video && this.video.storagePath) {
            this.videoPlayerUrl = this.video.storagePath;
        }
      },
      error: (err: Error) => {
        this.errorMessage = `Failed to load video details: ${err.message}`;
        this.isLoading = false;
      }
    });
  }

  saveChanges(): void { // Renamed from onSaveChanges to saveChanges for clarity
    if (!this.videoId || !this.editableVideo) {
      this.updateErrorMessage = 'No video data to update.';
      return;
    }

    this.isSavingMetadata = true;
    this.successMessage = '';
    this.updateErrorMessage = '';
    this.folderAssignmentMessage.text = '';


    const payload: UpdateVideoPayload = {
      title: this.editableVideo.title,
      description: this.editableVideo.description,
      tags: this.tagsString.split(',').map(tag => tag.trim()).filter(tag => tag !== ''),
      status: this.editableVideo.status
    };

    this.videoService.updateVideo(this.videoId, payload).subscribe({
      next: (updatedVideo: Video) => {
        this.video = updatedVideo;
        this.currentVideoTitle = updatedVideo.title;
        this.editableVideo = {
            title: updatedVideo.title,
            description: updatedVideo.description || '',
            tags: updatedVideo.tags || [],
            status: updatedVideo.status
        };
        this.tagsString = (updatedVideo.tags || []).join(', ');
        // Note: selectedFolderId is handled by assignToFolder method
        this.successMessage = 'Video metadata updated successfully!';
        this.isSavingMetadata = false;
      },
      error: (err: Error) => {
        this.updateErrorMessage = `Failed to update video metadata: ${err.message}`;
        this.isSavingMetadata = false;
      }
    });
  }

  assignToFolder(): void {
    if (!this.videoId) {
      this.folderAssignmentMessage = { type: 'error', text: 'Video ID is missing.' };
      return;
    }
    // selectedFolderId can be null (for unassigning) or a string ID.
    // Backend expects folderId: null or folderId: "actual_id".
    // Empty string "" from select [ngValue]="" should be converted to null if desired for unassignment.
    const targetFolderId = this.selectedFolderId === '' ? null : this.selectedFolderId;

    this.isAssigningFolder = true;
    this.successMessage = '';
    this.updateErrorMessage = '';
    this.folderAssignmentMessage.text = '';


    this.videoService.assignVideoToFolder(this.videoId, targetFolderId).subscribe({
      next: (updatedVideo: Video) => {
        this.video = updatedVideo; // Update video data, including its new folderId
        this.selectedFolderId = updatedVideo.folderId || null; // Re-sync dropdown
        this.folderAssignmentMessage = { type: 'success', text: 'Video folder updated successfully!' };
        this.isAssigningFolder = false;
      },
      error: (err: Error) => {
        this.folderAssignmentMessage = { type: 'error', text: `Failed to update video folder: ${err.message}` };
        this.isAssigningFolder = false;
      }
    });
  }


  goBackToList(): void {
    this.router.navigate(['/videos']);
  }

  ngOnDestroy(): void {
    if (this.routeSub) {
      this.routeSub.unsubscribe();
    }
  }
}
